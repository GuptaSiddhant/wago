package api

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

type inviteDTO struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	TeamID    string `json:"team_id,omitempty"`
	TeamName  string `json:"team_name,omitempty"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
	// Token is only echoed on creation so the admin can share the join link.
	Token string `json:"token,omitempty"`
}

func inviteToDTO(app core.App, inv *core.Record) inviteDTO {
	dto := inviteDTO{
		ID:        inv.Id,
		Email:     inv.GetString("email"),
		Role:      inv.GetString("role"),
		Status:    inv.GetString("status"),
		ExpiresAt: inv.GetDateTime("expires_at").Time().UTC().Format(types.DefaultDateLayout),
		CreatedAt: inv.GetDateTime("created").Time().UTC().Format(types.DefaultDateLayout),
	}
	if teamID := inv.GetString("team"); teamID != "" {
		dto.TeamID = teamID
		if team, err := app.FindRecordById("teams", teamID); err == nil {
			dto.TeamName = team.GetString("name")
		}
	}
	return dto
}

// HandleInviteCreate creates a pending invite for the given email. The invite
// token is returned so the inviter can share the join link with the recipient.
func HandleInviteCreate(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		if !access.CanManage() {
			return e.ForbiddenError("your role cannot invite members", nil)
		}

		var body struct {
			Email  string `json:"email" form:"email"`
			Role   string `json:"role" form:"role"`
			TeamID string `json:"team_id" form:"team_id"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("Invalid request body", err)
		}
		body.Email = strings.ToLower(strings.TrimSpace(body.Email))
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
		teamID, err := resolveTeam(app, access, body.Role, body.TeamID)
		if err != nil {
			return e.BadRequestError(err.Error(), nil)
		}

		count, err := store.CountPendingInvitesForEmail(app, access.OrgID, body.Email)
		if err != nil {
			return e.InternalServerError("failed to check existing invites", err)
		}
		if count > 0 {
			return e.BadRequestError("this email already has a pending invite", nil)
		}

		// created_by is a relation to the users collection, and superusers are
		// not users records, so leave it unset when a superadmin sends invites.
		createdBy := e.Auth.Id
		if e.Auth.IsSuperuser() {
			createdBy = ""
		}

		inv, err := store.CreateInvite(app, access.OrgID, body.Email, body.Role, teamID, createdBy)
		if err != nil {
			return e.InternalServerError("failed to create invite", err)
		}

		dto := inviteToDTO(app, inv)
		dto.Token = inv.GetString("token")
		return e.JSON(http.StatusOK, dto)
	}
}

// HandleInvitesList lists invites for the current org.
func HandleInvitesList(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		records, err := store.FindOrgInvites(app, access.OrgID)
		if err != nil {
			return e.InternalServerError("failed to list invites", err)
		}

		items := make([]inviteDTO, 0, len(records))
		for _, r := range records {
			items = append(items, inviteToDTO(app, r))
		}
		return e.JSON(http.StatusOK, map[string]any{"items": items})
	}
}

// HandleInviteRevoke revokes a pending invite.
func HandleInviteRevoke(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		if !access.CanManage() {
			return e.ForbiddenError("your role cannot manage invites", nil)
		}

		inv, err := store.FindOrgRecord(app, access.OrgID, "invites", e.Request.PathValue("id"))
		if err != nil {
			return e.NotFoundError("invite not found", nil)
		}
		if inv.GetString("status") != store.InviteStatusPending {
			return e.BadRequestError("only pending invites can be revoked", nil)
		}

		inv.Set("status", store.InviteStatusRevoked)
		if err := app.Save(inv); err != nil {
			return e.InternalServerError("failed to revoke invite", err)
		}
		return e.JSON(http.StatusOK, map[string]any{"id": inv.Id, "status": store.InviteStatusRevoked})
	}
}

// inviteInfoDTO is non-sensitive invite metadata shown on the public join page.
type inviteInfoDTO struct {
	Email    string `json:"email"`
	Role     string `json:"role"`
	Status   string `json:"status"`
	OrgName  string `json:"org_name"`
	TeamName string `json:"team_name,omitempty"`
	Expired  bool   `json:"expired"`
}

// HandleInviteInfo returns public info about an invite by token, used by the
// join page to describe the org/role being joined.
func HandleInviteInfo(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		token := strings.TrimSpace(e.Request.URL.Query().Get("t"))
		if token == "" {
			return e.BadRequestError("token is required", nil)
		}

		inv, err := store.FindInviteByToken(app, token)
		if err != nil || inv.GetString("status") != store.InviteStatusPending {
			return e.BadRequestError("invalid or already used invite", nil)
		}
		if inv.GetDateTime("expires_at").Time().Before(time.Now()) {
			return e.BadRequestError("this invite has expired", nil)
		}

		dto := inviteInfoDTO{
			Email:  inv.GetString("email"),
			Role:   inv.GetString("role"),
			Status: inv.GetString("status"),
		}
		if org, err := app.FindRecordById("orgs", inv.GetString("org")); err == nil {
			dto.OrgName = org.GetString("name")
		}
		if teamID := inv.GetString("team"); teamID != "" {
			if team, err := app.FindRecordById("teams", teamID); err == nil {
				dto.TeamName = team.GetString("name")
			}
		}
		return e.JSON(http.StatusOK, dto)
	}
}

// HandleInviteAccept activates a pending invite: it creates (or links) a user
// account for the invited email, adds the org membership with the invite's
// role/team, and marks the invite accepted. This is a public endpoint secured
// by the invite token.
func HandleInviteAccept(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var body struct {
			Token    string `json:"token" form:"token"`
			Name     string `json:"name" form:"name"`
			Password string `json:"password" form:"password"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("Invalid request body", err)
		}
		body.Token = strings.TrimSpace(body.Token)
		body.Name = strings.TrimSpace(body.Name)
		if body.Token == "" {
			return e.BadRequestError("token is required", nil)
		}
		if len(body.Password) < 8 {
			return e.BadRequestError("password must be at least 8 characters", nil)
		}

		inv, err := store.FindInviteByToken(app, body.Token)
		if err != nil || inv.GetString("status") != store.InviteStatusPending {
			return e.BadRequestError("invalid or already used invite", nil)
		}
		if inv.GetDateTime("expires_at").Time().Before(time.Now()) {
			return e.BadRequestError("this invite has expired", nil)
		}

		orgID := inv.GetString("org")
		email := strings.ToLower(strings.TrimSpace(inv.GetString("email")))

		user, err := store.FindUserByEmail(app, email)
		if err != nil {
			user, err = store.CreateUser(app, email, body.Name, body.Password)
			if err != nil {
				return e.InternalServerError("failed to create user account", err)
			}
		} else {
			if body.Name != "" {
				user.Set("name", body.Name)
			}
			user.Set("password", body.Password)
			user.Set("passwordConfirm", body.Password)
			if err := app.Save(user); err != nil {
				return e.InternalServerError("failed to update user account", err)
			}
		}

		member, err := store.EnsureOrgMember(app, orgID, user.Id, inv.GetString("role"))
		if err != nil {
			return e.InternalServerError("failed to add membership", err)
		}
		if teamID := inv.GetString("team"); teamID != "" {
			member.Set("team", teamID)
			if err := app.Save(member); err != nil {
				return e.InternalServerError("failed to set membership team", err)
			}
		}

		inv.Set("status", store.InviteStatusAccepted)
		if err := app.Save(inv); err != nil {
			return e.InternalServerError("failed to finalize invite", err)
		}

		return e.JSON(http.StatusOK, map[string]any{"email": email})
	}
}
