package store

import (
	"crypto/rand"
	"fmt"
	"log"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleAgent  = "agent"
	RoleViewer = "viewer"
)

// valid roles in ascending access order
var AllRoles = []string{RoleOwner, RoleAdmin, RoleAgent, RoleViewer}

func EnsureOrgsCollection(app core.App) error {
	if _, err := app.FindCollectionByNameOrId("orgs"); err != nil {
		col := core.NewBaseCollection("orgs")
		// Orgs are only reachable through the scoped API (and PB superusers
		// which always bypass rules). Hidden from regular raw API access so
		// users can't enumerate other organizations.
		col.ListRule = nil
		col.ViewRule = nil
		col.CreateRule = nil
		col.UpdateRule = nil
		col.DeleteRule = nil

		col.Fields.Add(
			&core.TextField{Name: "name", Required: true},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		if err := app.Save(col); err != nil {
			return fmt.Errorf("failed to auto-create orgs collection: %w", err)
		}
		log.Println("Auto-created 'orgs' collection")
	}

	return nil
}

func EnsureOrgMembersCollection(app core.App) error {
	orgsCol, err := app.FindCollectionByNameOrId("orgs")
	if err != nil {
		return fmt.Errorf("orgs collection not found: %w", err)
	}
	usersCol, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return fmt.Errorf("users collection not found: %w", err)
	}

	if _, err := app.FindCollectionByNameOrId("org_members"); err != nil {
		collection := core.NewBaseCollection("org_members")

		collection.ListRule = types.Pointer("user = @request.auth.id")
		collection.ViewRule = types.Pointer("user = @request.auth.id")
		collection.CreateRule = nil
		collection.UpdateRule = nil
		collection.DeleteRule = nil

		collection.Fields.Add(
			&core.RelationField{
				Name:          "org",
				CollectionId:  orgsCol.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.RelationField{
				Name:          "user",
				CollectionId:  usersCol.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.SelectField{
				Name:      "role",
				MaxSelect: 1,
				Required:  true,
				Values:    AllRoles,
			},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		collection.AddIndex("idx_org_members_unique", true, "org, user", "")

		if err := app.Save(collection); err != nil {
			return fmt.Errorf("failed to auto-create org_members collection: %w", err)
		}
		log.Println("Auto-created 'org_members' collection")
	}

	return nil
}

// EnsureOrgMember makes sure the given user is a member of the given org.
// Returns the (possibly new) org_members record.
func EnsureOrgMember(app core.App, orgID, userID, role string) (*core.Record, error) {
	member, err := app.FindFirstRecordByFilter("org_members",
		"org = {:org} && user = {:user}",
		dbx.Params{"org": orgID, "user": userID})
	if err == nil {
		if member.GetString("role") != role && role != "" {
			member.Set("role", role)
			if err := app.Save(member); err != nil {
				return nil, fmt.Errorf("failed to update org_members role: %w", err)
			}
		}
		return member, nil
	}

	membersCol, err := app.FindCollectionByNameOrId("org_members")
	if err != nil {
		return nil, err
	}
	member = core.NewRecord(membersCol)
	member.Set("org", orgID)
	member.Set("user", userID)
	if role == "" {
		role = RoleAgent
	}
	member.Set("role", role)
	if err := app.Save(member); err != nil {
		return nil, fmt.Errorf("failed to create org_members record: %w", err)
	}
	return member, nil
}

// FindOrgMembership returns the user's membership record for the given org.
func FindOrgMembership(app core.App, orgID, userID string) (*core.Record, error) {
	return app.FindFirstRecordByFilter("org_members",
		"org = {:org} && user = {:user}",
		dbx.Params{"org": orgID, "user": userID})
}

// FindUserByEmail returns a user record by email.
func FindUserByEmail(app core.App, email string) (*core.Record, error) {
	return app.FindAuthRecordByEmail("users", email)
}

// CreateUser creates a new user record with the given credentials.
func CreateUser(app core.App, email, name, password string) (*core.Record, error) {
	usersCol, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return nil, err
	}
	user := core.NewRecord(usersCol)
	user.Set("email", email)
	user.Set("password", password)
	user.Set("passwordConfirm", password)
	if name != "" {
		user.Set("name", name)
	}
	if err := app.Save(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return user, nil
}

// CountRoleMembers returns how many members of the org hold the given role.
func CountRoleMembers(app core.App, orgID, role string) (int, error) {
	records, err := app.FindRecordsByFilter("org_members",
		"org = {:org} && role = {:role}", "", 500, 0,
		DbxParams(map[string]any{"org": orgID, "role": role}))
	if err != nil {
		return 0, err
	}
	return len(records), nil
}

// GeneratePassword returns a random 16-character password for new users.
func GeneratePassword() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("wago-%d", time.Now().UnixNano())
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}
