package api

import (
	"net/http"
	"time"

	"github.com/guptasiddhant/wago/pkg/meta"
	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
)

// serviceWindow is how long after the last inbound message we can reply
// with a free-form message (Meta customer service window).
const serviceWindow = 24 * time.Hour

type templateRequest struct {
	Name       string           `json:"name"`
	Language   string           `json:"language"`
	Parameters []map[string]any `json:"parameters,omitempty"`
}

type sendMessageRequest struct {
	ConversationID string           `json:"conversation_id" form:"conversation_id"`
	Body           string           `json:"body" form:"body"`
	Template       *templateRequest `json:"template" form:"template"`
}

// HandleSendMessage sends an outbound WhatsApp message from the app.
func HandleSendMessage(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		var body sendMessageRequest
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("Invalid request body", err)
		}
		if body.ConversationID == "" {
			return e.BadRequestError("conversation_id is required", nil)
		}

		conv, err := app.FindRecordById("conversations", body.ConversationID)
		if err != nil {
			return e.NotFoundError("conversation not found", nil)
		}
		if conv.GetString("org") != access.OrgID {
			return e.ForbiddenError("you don't have access to this conversation", nil)
		}

		account, err := store.FindWhatsAppAccountByID(app, conv.GetString("whatsapp_account"))
		if err != nil {
			return e.InternalServerError("whatsapp account not found", err)
		}
		contact, err := app.FindRecordById("contacts", conv.GetString("contact"))
		if err != nil {
			return e.InternalServerError("contact not found", err)
		}

		to := contact.GetString("phone")
		accessToken := account.GetString("access_token")
		phoneNumberID := account.GetString("phone_number_id")

		// Determine if we're within the 24h customer service window.
		inWindow, err := isWithinServiceWindow(app, body.ConversationID)
		if err != nil {
			return e.InternalServerError("Failed to check service window", err)
		}

		// Enforce Meta policy: free-form only inside the window,
		// otherwise an approved template is required.
		if body.Template == nil && !inWindow {
			return e.BadRequestError(
				"The 24h customer service window has closed. Send an approved template message instead.", nil)
		}
		if body.Template != nil {
			if body.Body != "" {
				return e.BadRequestError("body must be empty when sending a template", nil)
			}
			if body.Template.Name == "" || body.Template.Language == "" {
				return e.BadRequestError("template name and language are required", nil)
			}
		}
		if body.Template == nil && body.Body == "" {
			return e.BadRequestError("message body is required", nil)
		}

		client := meta.NewClient()
		ctx := e.Request.Context()

		var wamid string
		var payload any
		if body.Template != nil {
			wamid, err = client.SendTemplate(ctx, accessToken, phoneNumberID, to,
				body.Template.Name, body.Template.Language, body.Template.Parameters)
			payload = map[string]any{"template": body.Template.Name, "language": body.Template.Language}
		} else {
			wamid, err = client.SendText(ctx, accessToken, phoneNumberID, to, body.Body)
			payload = map[string]any{"type": "text"}
		}
		if err != nil {
			return e.BadRequestError("Failed to send message via Meta", err)
		}

		msg, err := store.SaveOutgoingMessage(app, access.OrgID, body.ConversationID,
			phoneNumberID, to, body.Body, wamid, payload)
		if err != nil {
			return e.InternalServerError("Message sent but failed to save locally", err)
		}

		// bump conversation ordering
		conv.Set("last_message_at", time.Now())
		if err := app.Save(conv); err != nil {
			return e.InternalServerError("Failed to update conversation", err)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"id":        msg.Id,
			"wamid":     wamid,
			"status":    "sent",
			"in_window": inWindow,
		})
	}
}

// isWithinServiceWindow reports whether the last inbound message in the
// conversation happened within the 24h customer service window.
func isWithinServiceWindow(app core.App, conversationID string) (bool, error) {
	records, err := app.FindRecordsByFilter("messages",
		"conversation = {:conv} && direction = 'inbound'",
		"-created", 1, 0,
		store.DbxParams(map[string]any{"conv": conversationID}))
	if err != nil {
		return false, err
	}
	if len(records) == 0 {
		return false, nil
	}

	lastInbound := records[0].GetDateTime("created").Time()
	return time.Since(lastInbound) <= serviceWindow, nil
}
