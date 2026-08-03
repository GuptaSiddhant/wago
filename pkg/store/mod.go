package store

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// DbxParams converts a map[string]any into dbx.Params for use with
// the FindRecordsByFilter/FindFirstRecordByFilter filter builders.
func DbxParams(m map[string]any) dbx.Params {
	return dbx.Params(m)
}

func EnsureCollections(app core.App) error {
	if err := EnsureOrgsCollection(app); err != nil {
		return err
	}
	if err := EnsureOrgMembersCollection(app); err != nil {
		return err
	}
	if err := EnsureWhatsAppAccountsCollection(app); err != nil {
		return err
	}
	if err := EnsureContactsCollection(app); err != nil {
		return err
	}
	if err := EnsureConversationsCollection(app); err != nil {
		return err
	}
	if err := EnsureMessagesCollection(app); err != nil {
		return err
	}

	return nil
}
