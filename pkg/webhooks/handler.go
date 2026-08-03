package webhooks

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/guptasiddhant/wago/pkg/store"

	wawh "github.com/piusalfred/whatsapp/webhooks"
	"github.com/pocketbase/pocketbase/core"
)

// HandleVerification returns the handler for GET /api/wa/webhook
func HandleVerification(verifyToken string) func(re *core.RequestEvent) error {
	return func(re *core.RequestEvent) error {
		mode := re.Request.URL.Query().Get("hub.mode")
		token := re.Request.URL.Query().Get("hub.verify_token")
		challenge := re.Request.URL.Query().Get("hub.challenge")

		if mode == "subscribe" && token == verifyToken {
			log.Println(" Meta Webhook verified successfully!")
			return re.String(http.StatusOK, challenge)
		}

		return re.String(http.StatusForbidden, "Verification failed")
	}
}

// HandleIncomingMessage returns the handler for POST /api/wa/webhook.
// When appSecret is non-empty, inbound payloads are validated against the
// X-Hub-Signature-256 header signed with the Meta App Secret.
func HandleIncomingMessage(appSecret string) func(re *core.RequestEvent) error {
	return func(re *core.RequestEvent) error {
		notification, err := wawh.ExtractAndValidatePayload(re.Request, &wawh.ValidateOptions{
			Validate:  appSecret != "",
			AppSecret: appSecret,
		})
		if err != nil {
			log.Printf("Invalid webhook payload: %v", err)
			return re.String(http.StatusOK, "EVENT_RECEIVED")
		}

		processNotification(re.App, notification)

		// Always return 200 OK to Meta immediately
		return re.String(http.StatusOK, "EVENT_RECEIVED")
	}
}

func processNotification(app core.App, notification *wawh.Notification) {
	for _, entry := range notification.Entry {
		for _, change := range entry.Changes {
			processChange(app, change)
		}
	}
}

func processChange(app core.App, change wawh.Change) {
	val := change.Value
	if val == nil || val.Metadata == nil {
		return
	}

	// route to the org owning this phone number
	account, err := store.FindWhatsAppAccountByPhoneNumberID(app, val.Metadata.PhoneNumberID)
	if err != nil {
		log.Printf("Ignoring webhook for unknown phone_number_id %q", val.Metadata.PhoneNumberID)
		return
	}
	orgID := account.GetString("org")

	// delivery status updates (sent/delivered/read/failed)
	for _, st := range val.Statuses {
		if err := store.UpdateMessageStatus(app, st.ID, st.StatusValue); err != nil {
			log.Printf("Failed to update message status %q -> %s: %v", st.ID, st.StatusValue, err)
		}
	}

	// incoming messages
	if len(val.Messages) == 0 {
		return
	}

	senderName := "Unknown"
	waID := ""
	if len(val.Contacts) > 0 {
		if val.Contacts[0].Profile != nil {
			senderName = val.Contacts[0].Profile.Name
		}
		waID = val.Contacts[0].WaID
	}
	if waID == "" {
		return
	}

	contact, err := store.UpsertContact(app, orgID, waID, senderName)
	if err != nil {
		log.Printf("Failed to upsert contact: %v", err)
		return
	}

	ts := time.Now()
	conv, err := store.UpsertConversation(app, orgID, contact.Id, account.Id, ts)
	if err != nil {
		log.Printf("Failed to upsert conversation: %v", err)
		return
	}

	for _, msg := range val.Messages {
		if msg == nil || msg.Type != "text" || msg.Text == nil {
			// @todo handle media + template messages in a follow-up
			continue
		}

		msgTS := ts
		if sec, err := strconv.ParseInt(msg.Timestamp, 10, 64); err == nil {
			msgTS = time.Unix(sec, 0)
		}

		err := store.SaveIncomingMessage(
			app, orgID, conv.Id, msg.From, senderName,
			val.Metadata.DisplayPhoneNumber, msg.Text.Body, msg.ID, msgTS, val,
		)
		if err != nil {
			log.Printf("Failed to save message: %v", err)
			continue
		}

		// inbound messages count towards unread until an agent reads them
		_ = store.IncrementConversationUnread(app, conv.Id)
	}
}
