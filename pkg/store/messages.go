package store

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func EnsureMessagesCollection(app core.App) error {
	if _, err := app.FindCollectionByNameOrId("messages"); err != nil {
		collection := core.NewBaseCollection("messages")

		// Allow logged-in users to list, view, and create messages
		collection.ListRule = types.Pointer("@request.auth.id != ''")
		collection.ViewRule = types.Pointer("@request.auth.id != ''")
		collection.CreateRule = types.Pointer("@request.auth.id != ''")
		// Lock Update and Delete to Superusers only (nil = locked)
		collection.UpdateRule = nil
		collection.DeleteRule = nil

		collection.Fields.Add(
			&core.TextField{Name: "wamid", Required: true},
			&core.TextField{Name: "sender_phone", Required: true},
			&core.TextField{Name: "recipient_phone", Required: true},
			&core.TextField{Name: "body"},
			&core.SelectField{
				Name:      "direction",
				MaxSelect: 1,
				Values:    []string{"inbound", "outbound"},
			},
			&core.SelectField{
				Name:      "status",
				MaxSelect: 1,
				Values:    []string{"sent", "delivered", "read", "failed"},
			},
			&core.JSONField{Name: "payload", Required: true},

			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		// Add unique index on Meta's message ID (wamid)
		collection.AddIndex("idx_messages_wamid", true, "wamid", "")

		if err := app.Save(collection); err != nil {
			return fmt.Errorf("failed to auto-create messages collection: %w", err)
		}
		log.Println("Auto-created 'messages' collection")
	}

	return nil
}

// Helper to write incoming message & contact into PocketBase database
func SaveIncomingMessage(app core.App, senderPhone, senderName, recipientPhone, body, wamid string, payload any) error {
	messagesCol, err := app.FindCollectionByNameOrId("messages")
	if err != nil {
		return fmt.Errorf("messages collection not found: %w", err)
	}

	record := core.NewRecord(messagesCol)
	record.Set("wamid", wamid)
	record.Set("sender_phone", senderPhone)
	record.Set("recipient_phone", recipientPhone)
	record.Set("body", body)
	record.Set("direction", "inbound")
	record.Set("status", "read")
	record.Set("payload", payload)

	if err := app.Save(record); err != nil {
		return fmt.Errorf("failed to save message record: %w", err)
	}

	log.Printf(" Saved message from %s (%s): %s", senderName, senderPhone, body)
	return nil
}
