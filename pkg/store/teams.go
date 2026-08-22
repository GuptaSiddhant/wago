package store

import (
	"github.com/pocketbase/pocketbase/core"
)

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
