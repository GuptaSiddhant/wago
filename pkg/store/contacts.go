package store

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase/core"
)

// EnsureContactsCollection creates the org-scoped contacts collection that holds
// the people each org messages plus their tags/notes.
func EnsureContactsCollection(app core.App) error {
	orgsCol, err := app.FindCollectionByNameOrId("orgs")
	if err != nil {
		return fmt.Errorf("orgs collection not found: %w", err)
	}

	if _, err := app.FindCollectionByNameOrId("contacts"); err != nil {
		collection := core.NewBaseCollection("contacts")

		// Contacts are scoped to org_id and only reachable through the scoped
		// API (and PB superusers which always bypass rules). Hidden from
		// regular raw API access to prevent cross-org enumeration.
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
			&core.TextField{Name: "phone", Required: true},
			&core.TextField{Name: "name"},
			&core.JSONField{Name: "tags"},
			&core.TextField{Name: "notes"},
			&core.DateField{Name: "last_activity"},

			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		// Unique contact per org + phone
		collection.AddIndex("idx_contacts_org_phone", true, "org, phone", "")

		if err := app.Save(collection); err != nil {
			return fmt.Errorf("failed to auto-create contacts collection: %w", err)
		}
		log.Println("Auto-created 'contacts' collection")
	}

	return nil
}

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
