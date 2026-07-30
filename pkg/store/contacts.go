package store

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func EnsureContactsCollection(app core.App) error {
	if _, err := app.FindCollectionByNameOrId("contacts"); err != nil {
		collection := core.NewBaseCollection("contacts")

		// Allow logged-in users to list, view, and create contacts
		collection.ListRule = types.Pointer("@request.auth.id != ''")
		collection.ViewRule = types.Pointer("@request.auth.id != ''")
		collection.CreateRule = types.Pointer("@request.auth.id != ''")
		collection.UpdateRule = types.Pointer("@request.auth.id != ''")
		// Lock Delete to Superusers only (nil = locked)
		collection.DeleteRule = nil

		collection.Fields.Add(
			&core.TextField{Name: "phone", Required: true},
			&core.TextField{Name: "name"},

			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		// Add unique index on Meta's message ID (wamid)
		collection.AddIndex("idx_contacts_phone", true, "phone", "")

		if err := app.Save(collection); err != nil {
			return fmt.Errorf("failed to auto-create contacts collection: %w", err)
		}
		log.Println("Auto-created 'contacts' collection")
	}

	return nil
}
