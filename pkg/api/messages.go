package api

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/guptasiddhant/wago/pkg/meta"
	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
)

// serviceWindow is how long after the last inbound message we can reply
// with a free-form message (Meta customer service window).
const serviceWindow = 24 * time.Hour

// bumpConversationTimestamp refreshes a conversation's last_message_at so it
// jumps to the top of the inbox after an outbound send. Shared by all outbound
// handlers (text, media, template).
func bumpConversationTimestamp(app core.App, conv *core.Record) error {
	conv.Set("last_message_at", time.Now())
	return app.Save(conv)
}

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

		conv, err := store.FindOrgRecord(app, access.OrgID, "conversations", body.ConversationID)
		if err != nil {
			return e.NotFoundError("conversation not found", nil)
		}
		if !access.CanViewTeam(conv.GetString("team")) {
			return e.ForbiddenError("you don't have access to this conversation", nil)
		}

		account, err := store.FindWhatsAppAccountByID(app, access.OrgID, conv.GetString("whatsapp_account"))
		if err != nil {
			return e.InternalServerError("whatsapp account not found", err)
		}
		contact, err := store.FindOrgRecord(app, access.OrgID, "contacts", conv.GetString("contact"))
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
			header := findTemplateHeaderMedia(app, access.OrgID, body.Template.Name)
			wamid, err = client.SendTemplate(ctx, accessToken, phoneNumberID, to,
				body.Template.Name, body.Template.Language, body.Template.Parameters, header)
			payload = map[string]any{"template": body.Template.Name, "language": body.Template.Language}
		} else {
			wamid, err = client.SendText(ctx, accessToken, phoneNumberID, to, body.Body)
			payload = map[string]any{"type": "text"}
		}
		if err != nil {
			return e.BadRequestError("Failed to send message via Meta", err)
		}

		msg, err := store.SaveOutgoingMessage(app, access.OrgID, body.ConversationID,
			phoneNumberID, to, body.Body, wamid, payload, nil, "")
		if err != nil {
			return e.InternalServerError("Message sent but failed to save locally", err)
		}

		if err := bumpConversationTimestamp(app, conv); err != nil {
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

// HandleSendMediaMessage sends an outbound media message from the app. The
// request is a multipart form: file + conversation_id + optional caption.
// Media is free-form content, so the 24h customer service window policy
// applies exactly as it does for text messages.
func HandleSendMediaMessage(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		if err := e.Request.ParseMultipartForm(100 << 20); err != nil {
			return e.BadRequestError("Invalid multipart body", err)
		}

		conversationID := strings.TrimSpace(e.Request.FormValue("conversation_id"))
		if conversationID == "" {
			return e.BadRequestError("conversation_id is required", nil)
		}
		caption := strings.TrimSpace(e.Request.FormValue("caption"))

		file, header, err := e.Request.FormFile("file")
		if err != nil {
			return e.BadRequestError("file is required", nil)
		}
		defer file.Close()

		conv, err := store.FindOrgRecord(app, access.OrgID, "conversations", conversationID)
		if err != nil {
			return e.NotFoundError("conversation not found", nil)
		}
		if !access.CanViewTeam(conv.GetString("team")) {
			return e.ForbiddenError("you don't have access to this conversation", nil)
		}

		account, err := store.FindWhatsAppAccountByID(app, access.OrgID, conv.GetString("whatsapp_account"))
		if err != nil {
			return e.InternalServerError("whatsapp account not found", err)
		}
		contact, err := store.FindOrgRecord(app, access.OrgID, "contacts", conv.GetString("contact"))
		if err != nil {
			return e.InternalServerError("contact not found", err)
		}

		inWindow, err := isWithinServiceWindow(app, conversationID)
		if err != nil {
			return e.InternalServerError("Failed to check service window", err)
		}
		if !inWindow {
			return e.BadRequestError(
				"The 24h customer service window has closed. Send an approved template message instead.", nil)
		}

		kind := mediaKindForMime(header.Header.Get("Content-Type"))
		if kind == "" {
			return e.BadRequestError("unsupported file type; expected an image, video, audio, document or sticker", nil)
		}

		data, err := io.ReadAll(file)
		if err != nil {
			return e.InternalServerError("Failed to read file", err)
		}
		if len(data) == 0 {
			return e.BadRequestError("file is empty", nil)
		}

		to := contact.GetString("phone")
		accessToken := account.GetString("access_token")
		phoneNumberID := account.GetString("phone_number_id")

		client := meta.NewClient()
		ctx := e.Request.Context()

		mediaID, err := client.UploadMedia(ctx, accessToken, phoneNumberID, header.Filename, header.Header.Get("Content-Type"), data)
		if err != nil {
			return e.BadRequestError("Failed to upload media to Meta", err)
		}

		wamid, err := client.SendMediaByID(ctx, accessToken, phoneNumberID, to, kind, mediaID, caption, header.Filename)
		if err != nil {
			return e.BadRequestError("Failed to send media message via Meta", err)
		}

		payload := map[string]any{
			"type": kind,
			"media": map[string]any{
				"media_id": mediaID,
				"filename": header.Filename,
				"caption":  caption,
			},
		}
		msg, err := store.SaveOutgoingMessage(app, access.OrgID, conversationID,
			phoneNumberID, to, mediaCaptionText(kind, caption, header.Filename), wamid, payload,
			data, header.Filename)
		if err != nil {
			return e.InternalServerError("Message sent but failed to save locally", err)
		}

		if err := bumpConversationTimestamp(app, conv); err != nil {
			return e.InternalServerError("Failed to update conversation", err)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"id":        msg.Id,
			"wamid":     wamid,
			"status":    "sent",
			"kind":      kind,
			"in_window": inWindow,
		})
	}
}

// findTemplateHeaderMedia returns the send-time header media override for the
// org's template with the given name, or nil when the template has no media
// header. Media ids live on the local template record, so sending via name
// only (broadcasts, chat composer) can still attach the media.
func findTemplateHeaderMedia(app core.App, orgID, name string) *meta.TemplateHeaderMedia {
	record, err := store.FindOrgTemplateByName(app, orgID, name)
	if err != nil {
		return nil
	}
	return store.TemplateHeaderMedia(record)
}

// templateSendRequest is the body for sending a template to a contact to start
// (or continue) a conversation.
type templateSendRequest struct {
	ContactID  string           `json:"contact_id" form:"contact_id"`
	AccountID  string           `json:"account_id" form:"account_id"`
	TemplateID string           `json:"template_id" form:"template_id"`
	Parameters []map[string]any `json:"parameters" form:"parameters"`
}

// HandleTemplateSend sends an approved template to a contact, creating (or
// reusing) the conversation between the chosen number and the contact. This is
// the entry point that starts a chat from the contacts list or the composer.
func HandleTemplateSend(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		var body templateSendRequest
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("Invalid request body", err)
		}
		if body.ContactID == "" {
			return e.BadRequestError("contact_id is required", nil)
		}
		if body.AccountID == "" {
			return e.BadRequestError("account_id is required", nil)
		}
		if body.TemplateID == "" {
			return e.BadRequestError("template_id is required", nil)
		}

		contact, err := store.FindOrgRecord(app, access.OrgID, "contacts", body.ContactID)
		if err != nil {
			return e.NotFoundError("contact not found", nil)
		}
		if !access.CanViewTeam(contact.GetString("team")) {
			return e.ForbiddenError("you don't have access to this contact", nil)
		}

		account, err := store.FindOrgRecord(app, access.OrgID, "whatsapp_accounts", body.AccountID)
		if err != nil {
			return e.BadRequestError("whatsapp account not found", nil)
		}

		tmpl, err := store.FindOrgTemplate(app, access.OrgID, body.TemplateID)
		if err != nil {
			return e.NotFoundError("template not found", nil)
		}
		if !strings.EqualFold(tmpl.GetString("status"), "APPROVED") {
			return e.BadRequestError("only approved templates can be sent", nil)
		}

		to := contact.GetString("phone")
		accessToken := account.GetString("access_token")
		phoneNumberID := account.GetString("phone_number_id")
		if accessToken == "" || phoneNumberID == "" {
			return e.BadRequestError("this number needs an access token and phone number id to send messages", nil)
		}

		client := meta.NewClient()
		wamid, err := client.SendTemplate(e.Request.Context(), accessToken, phoneNumberID, to,
			tmpl.GetString("name"), tmpl.GetString("language"), body.Parameters,
			findTemplateHeaderMedia(app, access.OrgID, tmpl.GetString("name")))
		if err != nil {
			return e.BadRequestError("Failed to send template via Meta", err)
		}

		conv, _, err := store.UpsertConversation(app, access.OrgID, body.ContactID, body.AccountID, time.Now())
		if err != nil {
			return e.InternalServerError("Template sent but failed to create conversation", err)
		}

		payload := map[string]any{
			"template": tmpl.GetString("name"),
			"language": tmpl.GetString("language"),
		}
		msg, err := store.SaveOutgoingMessage(app, access.OrgID, conv.Id,
			phoneNumberID, to, "", wamid, payload, nil, "")
		if err != nil {
			return e.InternalServerError("Template sent but failed to save locally", err)
		}

		if err := bumpConversationTimestamp(app, conv); err != nil {
			return e.InternalServerError("Failed to update conversation", err)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"id":              msg.Id,
			"wamid":           wamid,
			"status":          "sent",
			"conversation_id": conv.Id,
		})
	}
}

// mediaKindForMime maps a browser-provided MIME type to a Cloud API media kind.
// Empty string means the file type is not supported for messaging.
func mediaKindForMime(mimeType string) string {
	if i := strings.Index(mimeType, ";"); i >= 0 {
		mimeType = strings.TrimSpace(mimeType[:i])
	}
	switch mimeType {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return meta.KindImage
	case "video/mp4", "video/3gpp", "video/quicktime":
		return meta.KindVideo
	case "audio/ogg", "audio/mpeg", "audio/mp3", "audio/amr", "audio/opus", "audio/mp4", "audio/aac", "audio/x-m4a":
		return meta.KindAudio
	case "application/pdf", "text/plain", "text/csv", "text/rtf", "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/vnd.ms-excel", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "application/vnd.ms-powerpoint", "application/vnd.openxmlformats-officedocument.presentationml.presentation", "application/zip", "application/x-zip-compressed":
		return meta.KindDocument
	default:
		return ""
	}
}

// mediaCaptionText renders the stored body for a media message: the caption
// when present, otherwise a short [Image]/[Video]/... label plus the filename.
func mediaCaptionText(kind, caption, filename string) string {
	if caption != "" {
		return caption
	}
	label := "Media"
	switch kind {
	case meta.KindImage:
		label = "[Image]"
	case meta.KindVideo:
		label = "[Video]"
	case meta.KindAudio:
		label = "[Audio]"
	case meta.KindDocument:
		label = "[Document]"
	case meta.KindSticker:
		label = "[Sticker]"
	}
	if filename != "" {
		return label + " " + filename
	}
	return label
}

// isWithinServiceWindow reports whether the last inbound message in the given
// conversation fell inside the 24h customer-service window, so replies are still
// allowed outside the 24h business-initiated-message limit.
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
