package store

import (
	"github.com/pocketbase/pocketbase/core"
)

// FindWhatsAppAccountByPhoneNumberID finds an account by its Meta phone_number_id.
func FindWhatsAppAccountByPhoneNumberID(app core.App, phoneNumberID string) (*core.Record, error) {
	return app.FindFirstRecordByFilter("whatsapp_accounts",
		"phone_number_id = {:phone_number_id}",
		DbxParams(map[string]any{"phone_number_id": phoneNumberID}))
}

// FindWhatsAppAccountByID finds an account by its record id.
func FindWhatsAppAccountByID(app core.App, orgID, id string) (*core.Record, error) {
	return FindOrgRecord(app, orgID, "whatsapp_accounts", id)
}
