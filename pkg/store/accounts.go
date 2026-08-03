package store

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func EnsureWhatsAppAccountsCollection(app core.App) error {
	orgsCol, err := app.FindCollectionByNameOrId("orgs")
	if err != nil {
		return fmt.Errorf("orgs collection not found: %w", err)
	}

	if _, err := app.FindCollectionByNameOrId("whatsapp_accounts"); err != nil {
		collection := core.NewBaseCollection("whatsapp_accounts")

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
			&core.TextField{Name: "display_name"},
			&core.TextField{Name: "phone_number_id", Required: true},
			&core.TextField{Name: "access_token", Required: true},
			&core.TextField{Name: "verify_token"},
			&core.SelectField{
				Name:      "status",
				MaxSelect: 1,
				Values:    []string{"connected", "disconnected"},
			},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		collection.AddIndex("idx_whatsapp_accounts_phone_number_id", true, "phone_number_id", "")

		if err := app.Save(collection); err != nil {
			return fmt.Errorf("failed to auto-create whatsapp_accounts collection: %w", err)
		}
		log.Println("Auto-created 'whatsapp_accounts' collection")
	}

	return nil
}

// FindWhatsAppAccountByPhoneNumberID finds an account by its Meta phone_number_id.
func FindWhatsAppAccountByPhoneNumberID(app core.App, phoneNumberID string) (*core.Record, error) {
	return app.FindFirstRecordByFilter("whatsapp_accounts",
		"phone_number_id = {:phone_number_id}",
		DbxParams(map[string]any{"phone_number_id": phoneNumberID}))
}

// FindWhatsAppAccountByID finds an account by its record id.
func FindWhatsAppAccountByID(app core.App, id string) (*core.Record, error) {
	return app.FindRecordById("whatsapp_accounts", id)
}
