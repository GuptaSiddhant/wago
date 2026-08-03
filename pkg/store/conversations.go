package store

import (
	"fmt"
	"log"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func EnsureConversationsCollection(app core.App) error {
	orgsCol, err := app.FindCollectionByNameOrId("orgs")
	if err != nil {
		return fmt.Errorf("orgs collection not found: %w", err)
	}
	contactsCol, err := app.FindCollectionByNameOrId("contacts")
	if err != nil {
		return fmt.Errorf("contacts collection not found: %w", err)
	}
	accountsCol, err := app.FindCollectionByNameOrId("whatsapp_accounts")
	if err != nil {
		return fmt.Errorf("whatsapp_accounts collection not found: %w", err)
	}
	usersCol, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return fmt.Errorf("users collection not found: %w", err)
	}

	if _, err := app.FindCollectionByNameOrId("conversations"); err != nil {
		collection := core.NewBaseCollection("conversations")

		collection.ListRule = types.Pointer("@request.auth.id != ''")
		collection.ViewRule = types.Pointer("@request.auth.id != ''")
		collection.CreateRule = types.Pointer("@request.auth.id != ''")
		collection.UpdateRule = types.Pointer("@request.auth.id != ''")
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
				Name:          "contact",
				CollectionId:  contactsCol.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.RelationField{
				Name:          "whatsapp_account",
				CollectionId:  accountsCol.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.RelationField{
				Name:         "assignee",
				CollectionId: usersCol.Id,
				MaxSelect:    1,
			},
			&core.NumberField{Name: "unread_count"},
			&core.DateField{Name: "last_message_at"},
			&core.SelectField{
				Name:      "status",
				MaxSelect: 1,
				Values:    []string{"open", "closed"},
			},

			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		// One conversation per (org, contact, whatsapp account)
		collection.AddIndex("idx_conversations_unique", true, "org, contact, whatsapp_account", "")

		if err := app.Save(collection); err != nil {
			return fmt.Errorf("failed to auto-create conversations collection: %w", err)
		}
		log.Println("Auto-created 'conversations' collection")
	}

	return nil
}

// UpsertConversation finds or creates the conversation for org+contact+account
// and bumps its last_message_at. Returns the conversation record.
func UpsertConversation(app core.App, orgID, contactID, accountID string, ts time.Time) (*core.Record, error) {
	conv, err := app.FindFirstRecordByFilter("conversations",
		"org = {:org} && contact = {:contact} && whatsapp_account = {:account}",
		DbxParams(map[string]any{"org": orgID, "contact": contactID, "account": accountID}))
	if err == nil {
		conv.Set("last_message_at", ts)
		if err := app.Save(conv); err != nil {
			return nil, fmt.Errorf("failed to update conversation: %w", err)
		}
		return conv, nil
	}

	convCol, err := app.FindCollectionByNameOrId("conversations")
	if err != nil {
		return nil, err
	}
	conv = core.NewRecord(convCol)
	conv.Set("org", orgID)
	conv.Set("contact", contactID)
	conv.Set("whatsapp_account", accountID)
	conv.Set("status", "open")
	conv.Set("unread_count", 0)
	conv.Set("last_message_at", ts)
	if err := app.Save(conv); err != nil {
		return nil, fmt.Errorf("failed to upsert conversation: %w", err)
	}
	return conv, nil
}

// IncrementConversationUnread increments the unread counter of a conversation.
func IncrementConversationUnread(app core.App, convID string) error {
	conv, err := app.FindRecordById("conversations", convID)
	if err != nil {
		return err
	}
	conv.Set("unread_count", conv.GetInt("unread_count")+1)
	return app.Save(conv)
}

// MarkConversationRead resets the unread counter to zero.
func MarkConversationRead(app core.App, convID string) error {
	conv, err := app.FindRecordById("conversations", convID)
	if err != nil {
		return err
	}
	conv.Set("unread_count", 0)
	return app.Save(conv)
}
