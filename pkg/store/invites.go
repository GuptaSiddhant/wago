package store

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

const inviteTTL = 7 * 24 * time.Hour

const (
	InviteStatusPending  = "pending"
	InviteStatusAccepted = "accepted"
	InviteStatusRevoked  = "revoked"
)

// EnsureInvitesCollection creates the org-scoped invites collection that holds
// pending/expired team-member onboarding invitations.
func EnsureInvitesCollection(app core.App) error {
	orgsCol, err := app.FindCollectionByNameOrId("orgs")
	if err != nil {
		return fmt.Errorf("orgs collection not found: %w", err)
	}
	teamsCol, err := app.FindCollectionByNameOrId("teams")
	if err != nil {
		return fmt.Errorf("teams collection not found: %w", err)
	}
	usersCol, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return fmt.Errorf("users collection not found: %w", err)
	}

	if _, err := app.FindCollectionByNameOrId("invites"); err != nil {
		col := core.NewBaseCollection("invites")
		// Invites are managed through the Wago API handlers only and scoped
		// to org_id (hidden from raw API access for non-superusers).
		col.ListRule = nil
		col.ViewRule = nil
		col.CreateRule = nil
		col.UpdateRule = nil
		col.DeleteRule = nil

		col.Fields.Add(
			&core.RelationField{
				Name:          "org",
				CollectionId:  orgsCol.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.RelationField{
				Name:         "team",
				CollectionId: teamsCol.Id,
				MaxSelect:    1,
			},
			&core.RelationField{
				Name:         "created_by",
				CollectionId: usersCol.Id,
				MaxSelect:    1,
			},
			&core.TextField{Name: "email", Required: true},
			&core.SelectField{Name: "role", MaxSelect: 1, Required: true, Values: AllRoles},
			&core.TextField{Name: "token", Required: true},
			&core.SelectField{
				Name:      "status",
				MaxSelect: 1,
				Required:  true,
				Values:    []string{InviteStatusPending, InviteStatusAccepted, InviteStatusRevoked},
			},
			&core.DateField{Name: "expires_at"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		col.AddIndex("idx_invites_token", true, "token", "")
		col.AddIndex("idx_invites_org_email", false, "org, email", "")

		if err := app.Save(col); err != nil {
			return fmt.Errorf("failed to auto-create invites collection: %w", err)
		}
		log.Println("Auto-created 'invites' collection")
	}

	return nil
}

// GenerateInviteToken returns a cryptographically random URL-safe token.
func GenerateInviteToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CreateInvite records a pending invite for an email with the given role/team.
func CreateInvite(app core.App, orgID, email, role, teamID, createdBy string) (*core.Record, error) {
	token, err := GenerateInviteToken()
	if err != nil {
		return nil, err
	}

	col, err := app.FindCollectionByNameOrId("invites")
	if err != nil {
		return nil, err
	}
	inv := core.NewRecord(col)
	inv.Set("org", orgID)
	inv.Set("email", email)
	inv.Set("role", role)
	if teamID != "" {
		inv.Set("team", teamID)
	}
	if createdBy != "" {
		inv.Set("created_by", createdBy)
	}
	inv.Set("token", token)
	inv.Set("status", InviteStatusPending)
	inv.Set("expires_at", time.Now().Add(inviteTTL))

	if err := app.Save(inv); err != nil {
		return nil, fmt.Errorf("failed to create invite: %w", err)
	}
	return inv, nil
}

// FindInviteByToken returns an invite by its accept token.
func FindInviteByToken(app core.App, token string) (*core.Record, error) {
	return app.FindFirstRecordByFilter("invites",
		"token = {:token}",
		DbxParams(map[string]any{"token": token}))
}

// FindOrgInvites lists invites of an org, most recent first.
func FindOrgInvites(app core.App, orgID string) ([]*core.Record, error) {
	return app.FindRecordsByFilter("invites",
		"org = {:org}", "-created", 200, 0,
		DbxParams(map[string]any{"org": orgID}))
}

// CountPendingInvitesForEmail returns how many pending invites exist for an
// org + email pair (used to prevent duplicates).
func CountPendingInvitesForEmail(app core.App, orgID, email string) (int, error) {
	records, err := app.FindRecordsByFilter("invites",
		"org = {:org} && email = {:email} && status = {:status}",
		"", 100, 0,
		DbxParams(map[string]any{"org": orgID, "email": email, "status": InviteStatusPending}))
	if err != nil {
		return 0, err
	}
	return len(records), nil
}
