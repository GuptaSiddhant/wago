package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

// AuthResult is the outcome of a successful login.
type AuthResult struct {
	Record  *core.Record
	Token   string
	IsAdmin bool // true when authenticated as a PocketBase superuser
}

// Authenticate validates credentials against the users collection first and
// falls back to the superusers collection. PocketBase superusers are
// super admins of the Wago UI as well.
func Authenticate(app core.App, email, password string) (*AuthResult, error) {
	// 1. try regular users
	if user, err := app.FindAuthRecordByEmail("users", email); err == nil {
		if !user.ValidatePassword(password) {
			return nil, errors.New("invalid credentials")
		}
		token, err := user.NewAuthToken()
		if err != nil {
			return nil, fmt.Errorf("failed to generate user auth token: %w", err)
		}
		return &AuthResult{Record: user, Token: token, IsAdmin: false}, nil
	}

	// 2. try superusers
	if su, err := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, email); err == nil {
		if !su.ValidatePassword(password) {
			return nil, errors.New("invalid credentials")
		}
		token, err := su.NewAuthToken()
		if err != nil {
			return nil, fmt.Errorf("failed to generate superuser auth token: %w", err)
		}
		return &AuthResult{Record: su, Token: token, IsAdmin: true}, nil
	}

	return nil, sql.ErrNoRows
}

// OrgAccess holds the resolved organization context for an authenticated request.
type OrgAccess struct {
	OrgID   string
	Role    string
	TeamID  string // empty for superadmins and org owners
	IsAdmin bool   // superuser bypasses membership checks
}

// ResolveOrgAccess verifies that the authenticated record can access the given org.
// Superusers can access any org. Regular users must be org members.
func ResolveOrgAccess(app core.App, auth *core.Record, orgID string) (*OrgAccess, error) {
	if auth == nil {
		return nil, errors.New("missing auth record")
	}

	if auth.IsSuperuser() {
		return &OrgAccess{OrgID: orgID, Role: RoleOwner, IsAdmin: true}, nil
	}

	member, err := FindOrgMembership(app, orgID, auth.Id)
	if err != nil {
		return nil, errors.New("you are not a member of this organization")
	}

	return &OrgAccess{
		OrgID:   orgID,
		Role:    member.GetString("role"),
		TeamID:  member.GetString("team"),
		IsAdmin: false,
	}, nil
}

// CanManage checks whether the given role can perform a management action
// (inviting members, managing teams, assigning conversations, etc.).
func (a *OrgAccess) CanManage() bool {
	return a.IsAdmin || a.Role == RoleOwner || a.Role == RoleAdmin
}

// CanManageData checks whether the role may mutate org data records
// (contacts, WhatsApp numbers). Superadmins and owners only.
func (a *OrgAccess) CanManageData() bool {
	return a.IsAdmin || a.Role == RoleOwner
}

// CanAssign checks whether the role may assign conversations to agents.
func (a *OrgAccess) CanAssign() bool {
	return a.IsAdmin || a.Role == RoleOwner || a.Role == RoleAdmin || a.Role == RoleAgent
}

// CanSeeAllConversations reports whether the role sees every conversation in
// the org regardless of team routing.
func (a *OrgAccess) CanSeeAllConversations() bool {
	return a.IsAdmin || a.Role == RoleOwner || a.Role == RoleAdmin
}

// CanViewTeam reports whether the member may view conversations routed to the
// given team. Agents and viewers only see their own team plus untagged
// conversations; owners/admins/superadmins see everything.
func (a *OrgAccess) CanViewTeam(teamID string) bool {
	if a.CanSeeAllConversations() {
		return true
	}
	return teamID == "" || teamID == a.TeamID
}

// UserOrgs returns the list of org memberships for a regular user.
func UserOrgs(app core.App, userID string) ([]*core.Record, error) {
	return app.FindRecordsByFilter("org_members", "user = {:user}", "-created", 50, 0,
		DbxParams(map[string]any{"user": userID}))
}

// AllOrgs returns every org (used by superusers).
func AllOrgs(app core.App) ([]*core.Record, error) {
	return app.FindRecordsByFilter("orgs", "", "-created", 200, 0)
}
