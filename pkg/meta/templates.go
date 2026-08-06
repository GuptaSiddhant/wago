package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// TemplateComponent is one part of a message template (header/body/footer/buttons).
type TemplateComponent struct {
	Type    string           `json:"type"`
	Format  string           `json:"format,omitempty"`
	Text    string           `json:"text,omitempty"`
	Example map[string]any   `json:"example,omitempty"`
	Buttons []TemplateButton `json:"buttons,omitempty"`
}

// TemplateButton is a quick-reply or call-to-action button on a template.
type TemplateButton struct {
	Type        string `json:"type"`
	Text        string `json:"text"`
	URL         string `json:"url,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
}

// TemplateSubmission is the payload for POST /{waba_id}/message_templates.
type TemplateSubmission struct {
	Name       string              `json:"name"`
	Language   string              `json:"language"`
	Category   string              `json:"category"`
	Components []TemplateComponent `json:"components"`
}

// TemplateMeta is a template as reported by the Graph API (list/detail).
type TemplateMeta struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Language       string              `json:"language"`
	Status         string              `json:"status"`
	Category       string              `json:"category"`
	RejectedReason string              `json:"rejected_reason"`
	Components     []TemplateComponent `json:"components"`
}

// CreateMessageTemplate submits a new template to the WhatsApp Business Account
// for review and returns the new template's Meta id.
func (c *Client) CreateMessageTemplate(ctx context.Context, accessToken, wabaID string, sub *TemplateSubmission) (string, error) {
	body, err := c.graphPost(ctx, accessToken, wabaID+"/message_templates", sub)
	if err != nil {
		return "", err
	}
	var resp struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("meta: parse create template response: %w", err)
	}
	if resp.ID == "" {
		return "", fmt.Errorf("meta: create template returned no id")
	}
	return resp.ID, nil
}

// ListMessageTemplates returns every template defined on the WABA, with their
// current review status.
func (c *Client) ListMessageTemplates(ctx context.Context, accessToken, wabaID string) ([]TemplateMeta, error) {
	fields := "id,name,language,status,category,rejected_reason,components"
	u := url.Values{}
	u.Set("fields", fields)
	u.Set("limit", "1000")
	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s/message_templates?%s", graphVersion, wabaID, u.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("meta: graph request: %w", err)
	}
	defer resp.Body.Close()

	body, err := readResponse(resp)
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Data []TemplateMeta `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("meta: parse template list: %w", err)
	}
	return parsed.Data, nil
}

// DeleteMessageTemplate deletes a template from the WABA by its Meta id.
func (c *Client) DeleteMessageTemplate(ctx context.Context, accessToken, wabaID, templateID string) error {
	_, err := c.graphDelete(ctx, accessToken, wabaID+"/message_templates/"+templateID)
	return err
}

// graphPost performs an authenticated POST with a JSON body against the Graph
// API and returns the raw response body. Errors carry Meta's message.
func (c *Client) graphPost(ctx context.Context, accessToken, path string, payload any) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("meta: marshal graph payload: %w", err)
	}

	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s", graphVersion, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("meta: graph request: %w", err)
	}
	defer resp.Body.Close()

	body, err := readResponse(resp)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// graphDelete performs an authenticated DELETE against the Graph API.
func (c *Client) graphDelete(ctx context.Context, accessToken, path string) ([]byte, error) {
	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s", graphVersion, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("meta: graph request: %w", err)
	}
	defer resp.Body.Close()

	body, err := readResponse(resp)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// readResponse reads the Graph API body and turns non-2xx responses into errors
// carrying Meta's message.
func readResponse(resp *http.Response) ([]byte, error) {
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
