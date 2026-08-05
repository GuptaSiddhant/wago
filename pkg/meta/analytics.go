package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// PhoneNumberInfo is the health/status data available per phone number without
// extra permissions (uses the same per-account token as messaging).
type PhoneNumberInfo struct {
	ID                     string `json:"id"`
	DisplayPhoneNumber     string `json:"display_phone_number"`
	VerifiedName           string `json:"verified_name"`
	QualityRating          string `json:"quality_rating"`
	MessagingLimitTier     string `json:"messaging_limit_tier"`
	CodeVerificationStatus string `json:"code_verification_status"`
	Status                 string `json:"status"`
}

// AnalyticsDataPoint is one row of a conversation_analytics response: a count
// and cost for a phone number, category and type over a time window.
type AnalyticsDataPoint struct {
	Start                int64   `json:"start"`
	End                  int64   `json:"end"`
	Conversations        int64   `json:"conversation"`
	Cost                 float64 `json:"cost"`
	PhoneNumber          string  `json:"phone_number"`
	ConversationCategory string  `json:"conversation_category"`
	ConversationType     string  `json:"conversation_type"`
}

type graphError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    int    `json:"code"`
	} `json:"error"`
}

// GetPhoneNumberInfo fetches the health/status of a given phone number.
func (c *Client) GetPhoneNumberInfo(ctx context.Context, accessToken, phoneNumberID string) (*PhoneNumberInfo, error) {
	fields := "id,display_phone_number,verified_name,quality_rating,messaging_limit_tier,code_verification_status,status"
	body, err := c.graphGet(ctx, accessToken, phoneNumberID, fields)
	if err != nil {
		return nil, err
	}
	var info PhoneNumberInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("meta: parse phone number info: %w", err)
	}
	return &info, nil
}

// FetchConversationAnalytics queries a WhatsApp Business Account for
// conversation counts and costs over [start, end), broken down by phone
// number, category and type.
func (c *Client) FetchConversationAnalytics(ctx context.Context, accessToken, wabaID string, start, end int64, granularity string) ([]AnalyticsDataPoint, error) {
	fields := fmt.Sprintf(
		"conversation_analytics.start(%d).end(%d).granularity(%s).phone_numbers([]).dimensions([\"CONVERSATION_CATEGORY\",\"CONVERSATION_TYPE\",\"PHONE\"])",
		start, end, granularity,
	)
	body, err := c.graphGet(ctx, accessToken, wabaID, fields)
	if err != nil {
		return nil, err
	}

	var resp struct {
		ConversationAnalytics struct {
			Data []struct {
				DataPoints []AnalyticsDataPoint `json:"data_points"`
			} `json:"data"`
		} `json:"conversation_analytics"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("meta: parse conversation analytics: %w", err)
	}

	var out []AnalyticsDataPoint
	for _, d := range resp.ConversationAnalytics.Data {
		out = append(out, d.DataPoints...)
	}
	return out, nil
}

// graphGet performs an authenticated GET against the Meta Graph API and
// returns the raw response body. Non-2xx responses are turned into Go errors
// carrying Meta's error message (e.g. a missing permission).
func (c *Client) graphGet(ctx context.Context, accessToken, path, fields string) ([]byte, error) {
	u := url.Values{}
	u.Set("access_token", accessToken)
	u.Set("fields", fields)
	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s?%s", graphVersion, path, u.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("meta: graph request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("meta: read graph response: %w", err)
	}
	if resp.StatusCode >= 300 {
		var ge graphError
		if json.Unmarshal(body, &ge) == nil && ge.Error.Message != "" {
			return nil, fmt.Errorf("meta: %s", ge.Error.Message)
		}
		return nil, fmt.Errorf("meta: graph api returned %d", resp.StatusCode)
	}
	return body, nil
}
