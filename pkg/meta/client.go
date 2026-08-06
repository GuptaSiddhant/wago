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

// TemplateHeaderMedia describes a media override for a template's header
// component (image/video/document). When sending a template that has a media
// header, providing the media id here attaches/overrides it at send time.
type TemplateHeaderMedia struct {
	Kind    string // meta.KindImage, meta.KindVideo or meta.KindDocument
	MediaID string
}

// SendTemplate sends an approved template message and returns the Meta message
// id (wamid). params are raw Meta parameters (text/currency/etc.) for the
// template body. header, when non-nil, attaches a media component to the
// template's header for templates that carry a media header.
func (c *Client) SendTemplate(ctx context.Context, accessToken, phoneNumberID, to, name, language string, params []map[string]any, header *TemplateHeaderMedia) (string, error) {
	ctx = withCredentials(ctx, accessToken, phoneNumberID)

	tmpl, err := buildTemplate(name, language, params, header)
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

// buildTemplate builds a Cloud API template message with an optional media
// header and body components.
func buildTemplate(name, language string, params []map[string]any, header *TemplateHeaderMedia) (*message.Template, error) {
	tmpl := &message.Template{
		Name:     name,
		Language: &message.TemplateLanguage{Code: language},
	}

	if header != nil && header.MediaID != "" {
		param := &message.TemplateParameter{Type: header.Kind}
		switch header.Kind {
		case KindImage:
			param.Image = &message.Image{ID: header.MediaID}
		case KindVideo:
			param.Video = &message.Video{ID: header.MediaID}
		case KindDocument:
			param.Document = &message.Document{ID: header.MediaID}
		default:
			return nil, fmt.Errorf("meta: unsupported template header kind %q", header.Kind)
		}
		tmpl.Components = append(tmpl.Components, &message.TemplateComponent{
			Type:       message.TemplateComponentTypeHeader,
			Parameters: []*message.TemplateParameter{param},
		})
	}

	if len(params) == 0 {
		if len(tmpl.Components) == 0 {
			return tmpl, nil
		}
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
	tmpl.Components = append(tmpl.Components, comp)
	return tmpl, nil
}
