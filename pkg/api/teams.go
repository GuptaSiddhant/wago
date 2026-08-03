package api

import (
	"net/http"
	"strings"

	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
)

type teamDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	MemberCount int    `json:"member_count"`
}

// HandleTeamsList lists the teams of the current org.
func HandleTeamsList(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		teams, err := store.FindTeamsByOrg(app, access.OrgID)
		if err != nil {
			return e.InternalServerError("Failed to list teams", err)
		}

		items := make([]teamDTO, 0, len(teams))
		for _, t := range teams {
			count, err := store.CountTeamMembers(app, access.OrgID, t.Id)
			if err != nil {
				return e.InternalServerError("Failed to count team members", err)
			}
			items = append(items, teamDTO{ID: t.Id, Name: t.GetString("name"), MemberCount: count})
		}

		return e.JSON(http.StatusOK, map[string]any{"items": items})
	}
}

// HandleTeamCreate creates a team in the current org.
func HandleTeamsCreate(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		if !access.CanManage() {
			return e.ForbiddenError("your role cannot manage teams", nil)
		}

		var body struct {
			Name string `json:"name" form:"name"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("Invalid request body", err)
		}
		body.Name = strings.TrimSpace(body.Name)
		if body.Name == "" {
			return e.BadRequestError("name is required", nil)
		}

		teamsCol, err := app.FindCollectionByNameOrId("teams")
		if err != nil {
			return e.InternalServerError("teams collection not found", err)
		}
		team := core.NewRecord(teamsCol)
		team.Set("org", access.OrgID)
		team.Set("name", body.Name)
		if err := app.Save(team); err != nil {
			return e.BadRequestError("failed to create team", err)
		}

		return e.JSON(http.StatusOK, teamDTO{ID: team.Id, Name: body.Name})
	}
}

// HandleTeamUpdate renames a team in the current org.
func HandleTeamsUpdate(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		if !access.CanManage() {
			return e.ForbiddenError("your role cannot manage teams", nil)
		}

		team, err := store.FindTeamInOrg(app, access.OrgID, e.Request.PathValue("id"))
		if err != nil {
			return e.NotFoundError("team not found", nil)
		}

		var body struct {
			Name string `json:"name" form:"name"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("Invalid request body", err)
		}
		body.Name = strings.TrimSpace(body.Name)
		if body.Name == "" {
			return e.BadRequestError("name is required", nil)
		}

		team.Set("name", body.Name)
		if err := app.Save(team); err != nil {
			return e.BadRequestError("failed to rename team", err)
		}

		return e.JSON(http.StatusOK, teamDTO{ID: team.Id, Name: body.Name})
	}
}

// HandleTeamDelete deletes a team in the current org. Teams that still have
// members cannot be deleted (reassign them first). Conversations routed to the
// team are untagged and become visible to all members.
func HandleTeamsDelete(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		if !access.CanManage() {
			return e.ForbiddenError("your role cannot manage teams", nil)
		}

		team, err := store.FindTeamInOrg(app, access.OrgID, e.Request.PathValue("id"))
		if err != nil {
			return e.NotFoundError("team not found", nil)
		}

		count, err := store.CountTeamMembers(app, access.OrgID, team.Id)
		if err != nil {
			return e.InternalServerError("Failed to count team members", err)
		}
		if count > 0 {
			return e.BadRequestError("cannot delete a team that still has members; reassign them first", nil)
		}

		if err := store.ClearConversationsTeam(app, team.Id); err != nil {
			return e.InternalServerError("Failed to untag conversations", err)
		}

		if err := app.Delete(team); err != nil {
			return e.InternalServerError("Failed to delete team", err)
		}

		return e.JSON(http.StatusOK, map[string]any{"id": team.Id})
	}
}
