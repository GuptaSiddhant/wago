package store

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
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
	if err := EnsureNotificationsCollection(app); err != nil {
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
	if err := EnsurePresenceField(app); err != nil {
		return err
	}
	if err := EnsureUserPhoneField(app); err != nil {
		return err
	}

	// Enforce org data isolation on existing collections too (the Ensure*
	// functions above only set rules when a collection is first created, so
	// pre-existing databases would otherwise keep their old, looser rules).
	if err := EnforceCollectionSecurity(app); err != nil {
		return err
	}

	return nil
}

func setCollectionRules(app core.App, name string, listView string) error {
	col, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		return nil // collection not (yet) created — nothing to enforce
	}

	col.ListRule = nil
	col.ViewRule = nil
	col.CreateRule = nil
	col.UpdateRule = nil
	col.DeleteRule = nil

	if listView != "" {
		col.ListRule = types.Pointer(listView)
		col.ViewRule = types.Pointer(listView)
	}

	return app.Save(col)
}

// EnforceCollectionSecurity applies the org-isolation access rules to every
// org-scoped collection. org_members lets each user see only their own
// memberships; every other org-scoped collection is locked to superusers and
// the scoped Wago API handlers.
func EnforceCollectionSecurity(app core.App) error {
	for _, name := range []string{
		"orgs",
		"teams",
		"whatsapp_accounts",
		"contacts",
		"conversations",
		"messages",
		"invites",
	} {
		if err := setCollectionRules(app, name, ""); err != nil {
			return err
		}
	}
	// Users may only see their own account record through the raw API; this
	// prevents cross-user email enumeration while keeping auth flows intact
	// (all password/management flows run server-side, bypassing these rules).
	if err := setCollectionRules(app, "users", "id = @request.auth.id"); err != nil {
		return err
	}
	// org_members is the one place a rule can enforce org membership directly.
	return setCollectionRules(app, "org_members", "user = @request.auth.id")
}
