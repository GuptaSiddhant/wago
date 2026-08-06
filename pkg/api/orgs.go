package api

import (
	"net/http"
	"strings"

	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
)

// HandleOrgCreate lets a superadmin create an organization. Regular users
// cannot create orgs; they are invited into existing ones.
func HandleOrgCreate(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if !e.HasSuperuserAuth() {
			return e.ForbiddenError("only a superadmin can create organizations", nil)
		}

		var body struct {
			Name string `json:"name" form:"name"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("Invalid request body", err)
		}
		body.Name = strings.TrimSpace(body.Name)
		if body.Name == "" {
			return e.BadRequestError("organization name is required", nil)
		}

		col, err := app.FindCollectionByNameOrId("orgs")
		if err != nil {
			return e.InternalServerError("orgs collection not found", err)
		}
		org := core.NewRecord(col)
		org.Set("name", body.Name)
		if err := app.Save(org); err != nil {
			return e.BadRequestError("failed to create organization", err)
		}

		return e.JSON(http.StatusCreated, orgSummary{
			ID:   org.Id,
			Name: org.GetString("name"),
			Role: store.RoleOwner,
		})
	}
}
