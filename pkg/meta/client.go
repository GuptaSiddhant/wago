package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/piusalfred/whatsapp"
	"github.com/piusalfred/whatsapp/config"
	"github.com/piusalfred/whatsapp/message"
	whttp "github.com/piusalfred/whatsapp/pkg/http"
)

// graphVersion is the Meta Graph API version used for Cloud API calls.
const graphVersion = "v24.0"

// accountCredentials holds the per-account token and phone number id used to
// authorise a single outbound request. They travel through the request context
// and are resolved by the config reader on every send, which is what makes a
// single Client safe to share across all orgs and WhatsApp accounts.
type accountCredentials struct {
	accessToken   string
	phoneNumberID string
}

type credentialsKey struct{}

// Client is a multi-tenant wrapper around the piusalfred/whatsapp Cloud API
// message client.
type Client struct {
	base *message.BaseClient
}

// NewClient builds a Client backed by the piusalfred/whatsapp library.
// It is safe for concurrent use across all orgs and accounts.
func NewClient() *Client {
	sender := whttp.NewSender[message.Message](
		whttp.WithCoreClientHTTPClient[message.Message](&http.Client{Timeout: 30 * time.Second}),
	)
	base, _ := message.NewBaseClient(sender, config.ReaderFunc(readConfig))
	return &Client{base: base}
}

// readConfig resolves the credentials for the current send from the context.
func readConfig(ctx context.Context) (*config.Config, error) {
	creds, ok := ctx.Value(credentialsKey{}).(accountCredentials)
	if !ok {
		return nil, fmt.Errorf("meta: no account credentials in context")
	}
	return &config.Config{
		BaseURL:       whatsapp.BaseURL,
		APIVersion:    graphVersion,
		AccessToken:   creds.accessToken,
		PhoneNumberID: creds.phoneNumberID,
	}, nil
}

// SendText sends a free-form text message and returns the Meta message id (wamid).
func (c *Client) SendText(ctx context.Context, accessToken, phoneNumberID, to, body string) (string, error) {
	ctx = withCredentials(ctx, accessToken, phoneNumberID)

	resp, err := c.base.SendText(ctx, message.NewRequest(to, &message.Text{Body: body}))
	if err != nil {
		return "", err
	}
	if len(resp.Messages) == 0 {
		return "", fmt.Errorf("meta: no message id in response")
	}
	return resp.Messages[0].ID, nil
}

// SendTemplate sends an approved template message and returns the Meta message
// id (wamid). params are raw Meta parameters (text/currency/etc.) for the
// template body.
func (c *Client) SendTemplate(ctx context.Context, accessToken, phoneNumberID, to, name, language string, params []map[string]any) (string, error) {
	ctx = withCredentials(ctx, accessToken, phoneNumberID)

	tmpl, err := buildTemplate(name, language, params)
	if err != nil {
		return "", err
	}

	resp, err := c.base.SendTemplate(ctx, message.NewRequest(to, tmpl))
	if err != nil {
		return "", err
	}
	if len(resp.Messages) == 0 {
		return "", fmt.Errorf("meta: no message id in response")
	}
	return resp.Messages[0].ID, nil
}

func withCredentials(ctx context.Context, accessToken, phoneNumberID string) context.Context {
	return context.WithValue(ctx, credentialsKey{}, accountCredentials{
		accessToken:   accessToken,
		phoneNumberID: phoneNumberID,
	})
}

// buildTemplate builds a Cloud API template message with a body component.
func buildTemplate(name, language string, params []map[string]any) (*message.Template, error) {
	tmpl := &message.Template{
		Name:     name,
		Language: &message.TemplateLanguage{Code: language},
	}
	if len(params) == 0 {
		return tmpl, nil
	}

	comp := &message.TemplateComponent{Type: message.TemplateComponentTypeBody}
	for _, raw := range params {
		data, err := json.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("meta: marshal template parameter: %w", err)
		}
		var p message.TemplateParameter
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("meta: parse template parameter: %w", err)
		}
		comp.Parameters = append(comp.Parameters, &p)
	}
	tmpl.Components = []*message.TemplateComponent{comp}
	return tmpl, nil
}
