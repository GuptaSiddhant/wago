package store

import (
	"fmt"
	"log"

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
		col.ListRule = types.Pointer("@request.auth.id != ''")
		col.ViewRule = types.Pointer("@request.auth.id != ''")
		col.CreateRule = types.Pointer("@request.auth.id != ''")
		col.UpdateRule = types.Pointer("@request.auth.id != ''")
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

		collection.ListRule = types.Pointer("@request.auth.id != ''")
		collection.ViewRule = types.Pointer("@request.auth.id != ''")
		collection.CreateRule = types.Pointer("@request.auth.id != ''")
		collection.UpdateRule = types.Pointer("@request.auth.id != ''")
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
