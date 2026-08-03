package api

import (
	"net/http"
	"slices"
	"strings"

	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
)

// canManageOwner reports whether the actor may assign, change, or remove the
// owner role. Only superadmins and the org owner may touch the owner.
func canManageOwner(access *store.OrgAccess) bool {
	return access.IsAdmin || access.Role == store.RoleOwner
}

func teamMemberFromRecords(user, member *core.Record) teamMemberDTO {
	return teamMemberDTO{
		ID:    user.Id,
		Name:  user.GetString("name"),
		Email: user.Email(),
		Role:  member.GetString("role"),
	}
}

// HandleTeamCreate adds a user to the org as a member.
func HandleTeamCreate(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		if !access.CanManage() {
			return e.ForbiddenError("your role cannot manage team members", nil)
		}

		var body struct {
			Email string `json:"email" form:"email"`
			Name  string `json:"name" form:"name"`
			Role  string `json:"role" form:"role"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("Invalid request body", err)
		}
		body.Email = strings.TrimSpace(body.Email)
		body.Name = strings.TrimSpace(body.Name)
		if body.Email == "" {
			return e.BadRequestError("email is required", nil)
		}
		if body.Role == "" {
			body.Role = store.RoleAgent
		}
		if !slices.Contains(store.AllRoles, body.Role) {
			return e.BadRequestError("invalid role", nil)
		}
		if body.Role == store.RoleOwner && !canManageOwner(access) {
			return e.ForbiddenError("only the owner or a superadmin can assign the owner role", nil)
		}

		user, err := store.FindUserByEmail(app, body.Email)
		generatedPassword := ""
		if err != nil {
			generatedPassword = store.GeneratePassword()
			user, err = store.CreateUser(app, body.Email, body.Name, generatedPassword)
			if err != nil {
				return e.InternalServerError("failed to create user", err)
			}
		}

		member, err := store.EnsureOrgMember(app, access.OrgID, user.Id, body.Role)
		if err != nil {
			return e.InternalServerError("failed to add member", err)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"member":             teamMemberFromRecords(user, member),
			"generated_password": generatedPassword,
		})
	}
}

// HandleTeamUpdate changes a member's role (and optionally their display name).
func HandleTeamUpdate(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		if !access.CanManage() {
			return e.ForbiddenError("your role cannot manage team members", nil)
		}

		userID := e.Request.PathValue("id")
		member, err := store.FindOrgMembership(app, access.OrgID, userID)
		if err != nil {
			return e.NotFoundError("member not found", nil)
		}

		var body struct {
			Role string `json:"role" form:"role"`
			Name string `json:"name" form:"name"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("Invalid request body", err)
		}

		currentRole := member.GetString("role")
		if currentRole == store.RoleOwner && !canManageOwner(access) {
			return e.ForbiddenError("admins cannot modify the organization owner", nil)
		}

		if body.Role != "" {
			if !slices.Contains(store.AllRoles, body.Role) {
				return e.BadRequestError("invalid role", nil)
			}
			if body.Role == store.RoleOwner && !canManageOwner(access) {
				return e.ForbiddenError("only the owner or a superadmin can assign the owner role", nil)
			}
			if currentRole == store.RoleOwner && body.Role != store.RoleOwner {
				count, err := store.CountRoleMembers(app, access.OrgID, store.RoleOwner)
				if err != nil {
					return e.InternalServerError("failed to check owners", err)
				}
				if count <= 1 {
					return e.BadRequestError("cannot demote the last owner of the organization", nil)
				}
			}
			member.Set("role", body.Role)
		}

		user, err := app.FindRecordById("users", userID)
		if err != nil {
			return e.InternalServerError("user not found", err)
		}
		if body.Name != "" && body.Name != user.GetString("name") {
			user.Set("name", body.Name)
			if err := app.Save(user); err != nil {
				return e.InternalServerError("failed to update user", err)
			}
		}

		if err := app.Save(member); err != nil {
			return e.InternalServerError("failed to update member", err)
		}

		return e.JSON(http.StatusOK, teamMemberFromRecords(user, member))
	}
}

// HandleTeamDelete removes a member from the org.
func HandleTeamDelete(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		if !access.CanManage() {
			return e.ForbiddenError("your role cannot manage team members", nil)
		}

		userID := e.Request.PathValue("id")
		member, err := store.FindOrgMembership(app, access.OrgID, userID)
		if err != nil {
			return e.NotFoundError("member not found", nil)
		}

		if member.GetString("role") == store.RoleOwner && !canManageOwner(access) {
			return e.ForbiddenError("admins cannot remove the organization owner", nil)
		}
		if member.GetString("role") == store.RoleOwner {
			count, err := store.CountRoleMembers(app, access.OrgID, store.RoleOwner)
			if err != nil {
				return e.InternalServerError("failed to check owners", err)
			}
			if count <= 1 {
				return e.BadRequestError("cannot remove the last owner of the organization", nil)
			}
		}

		if err := app.Delete(member); err != nil {
			return e.InternalServerError("failed to remove member", err)
		}

		return e.JSON(http.StatusOK, map[string]any{"id": userID})
	}
}
