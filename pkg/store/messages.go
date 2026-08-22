package store

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// Helper to write incoming message into PocketBase database. When mediaData is
// non-empty the bytes are stored on the record's "media" file field so the
// inbox can render/download it without hitting Meta again.
func SaveIncomingMessage(app core.App, orgID, conversationID, senderPhone, senderName, recipientPhone, body, wamid string, ts any, payload any, mediaData []byte, mediaName string) error {
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
	if len(mediaData) > 0 {
		file, err := filesystem.NewFileFromBytes(mediaData, mediaName)
		if err != nil {
			return fmt.Errorf("failed to build media file: %w", err)
		}
		record.Set("media", file)
	}
	if ts != nil {
		record.Set("created", ts)
	}

	if err := app.Save(record); err != nil {
		return fmt.Errorf("failed to save message record: %w", err)
	}

	log.Printf("Saved message from %s (%s): %s", senderName, senderPhone, body)
	return nil
}

// SaveOutgoingMessage stores a sent message and returns its record. When
// mediaData is non-empty the bytes are stored on the record's "media" field.
func SaveOutgoingMessage(app core.App, orgID, conversationID, senderPhone, recipientPhone, body, wamid string, payload any, mediaData []byte, mediaName string) (*core.Record, error) {
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
	if len(mediaData) > 0 {
		file, err := filesystem.NewFileFromBytes(mediaData, mediaName)
		if err != nil {
			return nil, fmt.Errorf("failed to build media file: %w", err)
		}
		record.Set("media", file)
	}

	if err := app.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save outgoing message: %w", err)
	}
	return record, nil
}

// FindMessageByWamid fetches a message scoped to a single org by its Meta wamid.
func FindMessageByWamid(app core.App, orgID, wamid string) (*core.Record, error) {
	return app.FindFirstRecordByFilter("messages",
		"org = {:org} && wamid = {:wamid}",
		DbxParams(map[string]any{"org": orgID, "wamid": wamid}))
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
