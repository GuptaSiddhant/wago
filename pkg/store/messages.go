package store

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase/core"
)

func EnsureMessagesCollection(app core.App) error {
	orgsCol, err := app.FindCollectionByNameOrId("orgs")
	if err != nil {
		return fmt.Errorf("orgs collection not found: %w", err)
	}
	conversationsCol, err := app.FindCollectionByNameOrId("conversations")
	if err != nil {
		return fmt.Errorf("conversations collection not found: %w", err)
	}

	if _, err := app.FindCollectionByNameOrId("messages"); err != nil {
		collection := core.NewBaseCollection("messages")

		// Messages are scoped to org_id and only reachable through the scoped
		// API (and PB superusers which always bypass rules).
		collection.ListRule = nil
		collection.ViewRule = nil
		collection.CreateRule = nil
		collection.UpdateRule = nil
		collection.DeleteRule = nil

		collection.Fields.Add(
			&core.RelationField{
				Name:          "org",
				CollectionId:  orgsCol.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.RelationField{
				Name:          "conversation",
				CollectionId:  conversationsCol.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
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

// Helper to write incoming message into PocketBase database.
func SaveIncomingMessage(app core.App, orgID, conversationID, senderPhone, senderName, recipientPhone, body, wamid string, ts any, payload any) error {
	messagesCol, err := app.FindCollectionByNameOrId("messages")
	if err != nil {
		return fmt.Errorf("messages collection not found: %w", err)
	}

	record := core.NewRecord(messagesCol)
	record.Set("org", orgID)
	record.Set("conversation", conversationID)
	record.Set("wamid", wamid)
	record.Set("sender_phone", senderPhone)
	record.Set("recipient_phone", recipientPhone)
	record.Set("body", body)
	record.Set("direction", "inbound")
	record.Set("status", "read")
	record.Set("payload", payload)
	if ts != nil {
		record.Set("created", ts)
	}

	if err := app.Save(record); err != nil {
		return fmt.Errorf("failed to save message record: %w", err)
	}

	log.Printf("Saved message from %s (%s): %s", senderName, senderPhone, body)
	return nil
}

// SaveOutgoingMessage stores a sent message and returns its record.
func SaveOutgoingMessage(app core.App, orgID, conversationID, senderPhone, recipientPhone, body, wamid string, payload any) (*core.Record, error) {
	messagesCol, err := app.FindCollectionByNameOrId("messages")
	if err != nil {
		return nil, fmt.Errorf("messages collection not found: %w", err)
	}

	record := core.NewRecord(messagesCol)
	record.Set("org", orgID)
	record.Set("conversation", conversationID)
	record.Set("wamid", wamid)
	record.Set("sender_phone", senderPhone)
	record.Set("recipient_phone", recipientPhone)
	record.Set("body", body)
	record.Set("direction", "outbound")
	record.Set("status", "sent")
	record.Set("payload", payload)

	if err := app.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save outgoing message: %w", err)
	}
	return record, nil
}

// UpdateMessageStatus updates the delivery status of a message by its Meta wamid.
func UpdateMessageStatus(app core.App, wamid, status string) error {
	record, err := app.FindFirstRecordByFilter("messages",
		"wamid = {:wamid}",
		DbxParams(map[string]any{"wamid": wamid}))
	if err != nil {
		return fmt.Errorf("message %q not found: %w", wamid, err)
	}

	record.Set("status", status)
	return app.Save(record)
}
