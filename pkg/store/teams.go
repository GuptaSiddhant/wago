package store

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// ensureField adds a field to a collection if it doesn't already exist,
// allowing existing collections to be migrated in place.
func ensureField(col *core.Collection, field core.Field) {
	if col.Fields.GetByName(field.GetName()) == nil {
		col.Fields.Add(field)
	}
}

// EnsureTeamsCollection creates the org-scoped teams collection.
func EnsureTeamsCollection(app core.App) error {
	orgsCol, err := app.FindCollectionByNameOrId("orgs")
	if err != nil {
		return fmt.Errorf("orgs collection not found: %w", err)
	}

	col, err := app.FindCollectionByNameOrId("teams")
	if err != nil {
		col = core.NewBaseCollection("teams")
		col.ListRule = types.Pointer("@request.auth.id != ''")
		col.ViewRule = types.Pointer("@request.auth.id != ''")
		col.CreateRule = types.Pointer("@request.auth.id != ''")
		col.UpdateRule = types.Pointer("@request.auth.id != ''")
		col.DeleteRule = nil

		col.Fields.Add(
			&core.RelationField{
				Name:          "org",
				CollectionId:  orgsCol.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.TextField{Name: "name", Required: true},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		col.AddIndex("idx_teams_org_name", true, "org, name", "")

		if err := app.Save(col); err != nil {
			return fmt.Errorf("failed to auto-create teams collection: %w", err)
		}
		log.Println("Auto-created 'teams' collection")
	}

	return nil
}

// EnsureTeamReferenceField adds the nullable team relation to a collection
// whose records belong to an org (org_members, whatsapp_accounts, conversations).
func EnsureTeamReferenceField(app core.App, collectionName string) error {
	teamsCol, err := app.FindCollectionByNameOrId("teams")
	if err != nil {
		return fmt.Errorf("teams collection not found: %w", err)
	}

	col, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return fmt.Errorf("%s collection not found: %w", collectionName, err)
	}

	ensureField(col, &core.RelationField{
		Name:         "team",
		CollectionId: teamsCol.Id,
		MaxSelect:    1,
	})

	if err := app.Save(col); err != nil {
		return fmt.Errorf("failed to add team field to %s: %w", collectionName, err)
	}
	return nil
}

// FindTeamsByOrg returns the teams of an org, most recently created last.
func FindTeamsByOrg(app core.App, orgID string) ([]*core.Record, error) {
	return app.FindRecordsByFilter("teams", "org = {:org}", "created", 100, 0,
		DbxParams(map[string]any{"org": orgID}))
}

// FindTeamInOrg returns a team belonging to the given org, or an error.
func FindTeamInOrg(app core.App, orgID, teamID string) (*core.Record, error) {
	return app.FindFirstRecordByFilter("teams",
		"org = {:org} && id = {:id}",
		DbxParams(map[string]any{"org": orgID, "id": teamID}))
}

// CountTeamMembers returns how many org members are assigned to the team.
func CountTeamMembers(app core.App, orgID, teamID string) (int, error) {
	records, err := app.FindRecordsByFilter("org_members",
		"org = {:org} && team = {:team}", "", 500, 0,
		DbxParams(map[string]any{"org": orgID, "team": teamID}))
	if err != nil {
		return 0, err
	}
	return len(records), nil
}

// ClearConversationsTeam unassigns a team from all its conversations.
func ClearConversationsTeam(app core.App, teamID string) error {
	records, err := app.FindRecordsByFilter("conversations",
		"team = {:team}", "", 1000, 0,
		DbxParams(map[string]any{"team": teamID}))
	if err != nil {
		return err
	}
	for _, r := range records {
		r.Set("team", "")
		if err := app.Save(r); err != nil {
			return err
		}
	}
	return nil
}
