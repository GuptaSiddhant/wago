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
	if err := EnsureTeamsCollection(app); err != nil {
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
	if err := EnsureInvitesCollection(app); err != nil {
		return err
	}
	// The team relation must exist on org_members before any member is created,
	// and on whatsapp_accounts/conversations for team-scoped routing.
	if err := EnsureTeamReferenceField(app, "org_members"); err != nil {
		return err
	}
	if err := EnsureTeamReferenceField(app, "whatsapp_accounts"); err != nil {
		return err
	}
	if err := EnsureTeamReferenceField(app, "conversations"); err != nil {
		return err
	}

	return nil
}
