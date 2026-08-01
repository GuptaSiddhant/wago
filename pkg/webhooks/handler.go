package webhooks

import (
	"log"
	"net/http"

	"guptasiddhant/wago/pkg/store"

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

// HandleIncomingMessage returns the handler for POST /api/wa/webhook
func HandleIncomingMessage() func(re *core.RequestEvent) error {
	return func(re *core.RequestEvent) error {
		var payload MetaWebhookPayload
		if err := re.BindBody(&payload); err != nil {
			return re.BadRequestError("Invalid JSON payload", err)
		}

		// Process entries
		for _, entry := range payload.Entry {
			for _, change := range entry.Changes {
				val := change.Value

				senderName := "Unknown"
				if len(val.Contacts) > 0 {
					senderName = val.Contacts[0].Profile.Name
				}

				for _, msg := range val.Messages {
					if msg.Type == "text" {
						// Save record using store layer
						err := store.SaveIncomingMessage(
							re.App, msg.From, senderName, val.Metadata.DisplayPhoneNumber, msg.Text.Body, msg.ID, val,
						)
						if err != nil {
							log.Printf("Failed to save message: %v", err)
						}
					}
				}
			}
		}

		// Always return 200 OK to Meta immediately
		return re.String(http.StatusOK, "EVENT_RECEIVED")
	}
}
