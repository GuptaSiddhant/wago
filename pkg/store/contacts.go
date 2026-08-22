package store

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

// UpsertContact finds or creates a contact for the given org+phone.
func UpsertContact(app core.App, orgID, phone, name string) (*core.Record, error) {
	contact, err := app.FindFirstRecordByFilter("contacts",
		"org = {:org} && phone = {:phone}",
		DbxParams(map[string]any{"org": orgID, "phone": phone}))
	if err == nil {
		return contact, nil
	}

	contactsCol, err := app.FindCollectionByNameOrId("contacts")
	if err != nil {
		return nil, err
	}
	contact = core.NewRecord(contactsCol)
	contact.Set("org", orgID)
	contact.Set("phone", phone)
	if name != "" {
		contact.Set("name", name)
	}
	if err := app.Save(contact); err != nil {
		return nil, fmt.Errorf("failed to upsert contact: %w", err)
	}
	return contact, nil
}

// FindContactByPhone returns a contact by org and phone.
func FindContactByPhone(app core.App, orgID, phone string) (*core.Record, error) {
	return app.FindFirstRecordByFilter("contacts",
		"org = {:org} && phone = {:phone}",
		DbxParams(map[string]any{"org": orgID, "phone": phone}))
}
