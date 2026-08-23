package store

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

// saveUser creates an auth user record for assignment tests.
func saveUser(t *testing.T, app core.App, email string) *core.Record {
	t.Helper()
	usersCol, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("users collection: %v", err)
	}
	u := core.NewRecord(usersCol)
	u.Set("email", email)
	u.Set("password", "supersecret123")
	u.Set("passwordConfirm", "supersecret123")
	if err := app.Save(u); err != nil {
		t.Fatalf("save user %s: %v", email, err)
	}
	return u
}

func TestAssignConversationRRExcludesViewersAndBalancesLoad(t *testing.T) {
	app := setupApp(t)

	org := saveRecord(t, app, "orgs", map[string]any{"name": "acme"})
	acc := saveRecord(t, app, "whatsapp_accounts", map[string]any{
		"org": org.Id, "display_name": "Sales", "phone_number_id": "111", "access_token": "tok",
	})
	c := saveRecord(t, app, "contacts", map[string]any{"org": org.Id, "name": "Ada", "phone": "5550001"})

	owner := saveUser(t, app, "owner@example.com")
	agent := saveUser(t, app, "agent@example.com")
	viewer := saveUser(t, app, "viewer@example.com")

	saveRecord(t, app, "org_members", map[string]any{"org": org.Id, "user": owner.Id, "role": RoleOwner})
	saveRecord(t, app, "org_members", map[string]any{"org": org.Id, "user": agent.Id, "role": RoleAgent})
	saveRecord(t, app, "org_members", map[string]any{"org": org.Id, "user": viewer.Id, "role": RoleViewer})

	eligible := map[string]bool{owner.Id: true, agent.Id: true}

	conv1 := saveRecord(t, app, "conversations", map[string]any{
		"org": org.Id, "contact": c.Id, "whatsapp_account": acc.Id,
	})
	a1, err := AssignConversationRR(app, conv1)
	if err != nil {
		t.Fatalf("AssignConversationRR #1: %v", err)
	}
	if !eligible[a1] {
		t.Fatalf("assignee %q is not an eligible agent/owner", a1)
	}

	// Round-robin must now prefer the other eligible member (least load).
	pick, err := PickRoundRobinAssignee(app, org.Id, "")
	if err != nil {
		t.Fatalf("PickRoundRobinAssignee: %v", err)
	}
	if !eligible[pick] {
		t.Fatalf("pick %q is not eligible", pick)
	}
}

func TestPickRoundRobinAssigneeErrorsWithoutMembers(t *testing.T) {
	app := setupApp(t)
	org := saveRecord(t, app, "orgs", map[string]any{"name": "empty org"})
	if _, err := PickRoundRobinAssignee(app, org.Id, ""); err == nil {
		t.Fatal("expected error when org has no members")
	}
}
