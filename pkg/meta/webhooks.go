package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// webhookFields are the Cloud API event types the Wago webhook consumes.
// Subscribe the app to these so inbound messages are delivered.
var webhookFields = []string{"messages", "message_template_status_update"}

// SubscribeWebhook subscribes the current Meta app to the webhook events of a
// WhatsApp Business Account. CallbackURL and verifyToken configure where and
// how Meta delivers the events. Both parameters come from the Wago instance:
// callbackURL is the public /api/wa/webhook endpoint and verifyToken matches
// the one validated in HandleVerification.
//
// The account's access token must belong to the same app that owns the WABA
// and carry the whatsapp_business_messaging scope.
func (c *Client) SubscribeWebhook(ctx context.Context, accessToken, wabaID, callbackURL, verifyToken string) error {
	// The callback and verify token are set per app (not per number). Set them
	// first so a successful subscription immediately starts delivering.
	if err := c.configureWebhook(ctx, accessToken, wabaID, callbackURL, verifyToken); err != nil {
		return err
	}

	// Subscribe the app to this WABA's webhook events.
	form := url.Values{}
	form.Set("access_token", accessToken)
	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s/subscribed_apps", graphVersion, wabaID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return err
	}
	req.URL.RawQuery = form.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("meta: subscribe webhook: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		var ge graphError
		if json.Unmarshal(body, &ge) == nil && ge.Error.Message != "" {
			return fmt.Errorf("meta: %s", ge.Error.Message)
		}
		return fmt.Errorf("meta: subscribe webhook returned %d", resp.StatusCode)
	}

	var out struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(body, &out); err == nil && !out.Success {
		return fmt.Errorf("meta: subscribe webhook was not acknowledged by Meta")
	}

	return nil
}

// configureWebhook registers the app-level callback URL and verify token for a
// WABA. The token must have whatsapp_business_messaging scope.
func (c *Client) configureWebhook(ctx context.Context, accessToken, wabaID, callbackURL, verifyToken string) error {
	form := url.Values{}
	form.Set("access_token", accessToken)
	form.Set("callback_url", callbackURL)
	form.Set("verify_token", verifyToken)
	form.Set("fields", joinFields(webhookFields))

	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s/config", graphVersion, wabaID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("meta: configure webhook: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		var ge graphError
		if json.Unmarshal(body, &ge) == nil && ge.Error.Message != "" {
			return fmt.Errorf("meta: %s", ge.Error.Message)
		}
		return fmt.Errorf("meta: configure webhook returned %d", resp.StatusCode)
	}

	var out struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(body, &out); err == nil && !out.Success {
		return fmt.Errorf("meta: webhook configuration was not acknowledged by Meta")
	}

	return nil
}

func joinFields(fields []string) string {
	out := ""
	for i, f := range fields {
		if i > 0 {
			out += ","
		}
		out += f
	}
	return out
}
