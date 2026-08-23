package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
)

// setupAPIApp builds a test app with collections, one org, and an admin-role
// user who is a member of that org.
func setupAPIApp(t *testing.T) (*tests.TestApp, *core.Record, *core.Record) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	if _, err := app.FindCollectionByNameOrId("users"); err != nil {
		if err := app.Save(core.NewAuthCollection("users")); err != nil {
			t.Fatalf("failed to create users collection: %v", err)
		}
	}
	if err := store.EnsureCollections(app); err != nil {
		t.Fatalf("failed to ensure collections: %v", err)
	}

	org := saveAPIRecord(t, app, "orgs", map[string]any{"name": "acme"})
	user := saveAPIUser(t, app, "agent@example.com")
	saveAPIRecord(t, app, "org_members", map[string]any{
		"org": org.Id, "user": user.Id, "role": store.RoleAdmin,
	})

	return app, org, user
}

func saveAPIUser(t *testing.T, app core.App, email string) *core.Record {
	t.Helper()
	usersCol, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("users collection: %v", err)
	}
	u := core.NewRecord(usersCol)
	u.Set("email", email)
	u.Set("password", "supersecret123")
	u.Set("passwordConfirm", "supersecret123")
	u.Set("name", "Agent")
	if err := app.Save(u); err != nil {
		t.Fatalf("save user: %v", err)
	}
	return u
}

func saveAPIRecord(t *testing.T, app core.App, col string, vals map[string]any) *core.Record {
	t.Helper()
	c, err := app.FindCollectionByNameOrId(col)
	if err != nil {
		t.Fatalf("find %s: %v", col, err)
	}
	rec := core.NewRecord(c)
	for k, v := range vals {
		rec.Set(k, v)
	}
	if err := app.Save(rec); err != nil {
		t.Fatalf("save %s: %v", col, err)
	}
	return rec
}

// newEvent builds a synthetic request event; auth mirrors what the RequireAuth
// middleware would inject before our handlers run.
func newEvent(app core.App, auth *core.Record, method, target, body string) *core.RequestEvent {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ev := &core.RequestEvent{App: app, Auth: auth}
	ev.Request = req
	ev.Response = httptest.NewRecorder()
	return ev
}

func TestOrgAccessRequiresHeader(t *testing.T) {
	app, _, user := setupAPIApp(t)

	e := newEvent(app, user, http.MethodGet, "/api/wa/inbox", "")
	_, apiErr := orgAccessFromRequest(e, app)
	if apiErr == nil || apiErr.Status != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing X-Org-Id, got %+v", apiErr)
	}
}

func TestOrgAccessRejectsNonMember(t *testing.T) {
	app, _, _ := setupAPIApp(t)
	outsider := saveAPIUser(t, app, "outsider@example.com")
	orgB := saveAPIRecord(t, app, "orgs", map[string]any{"name": "other org"})

	e := newEvent(app, outsider, http.MethodGet, "/api/wa/inbox", "")
	e.Request.Header.Set("X-Org-Id", orgB.Id)

	_, apiErr := orgAccessFromRequest(e, app)
	if apiErr == nil || apiErr.Status != http.StatusForbidden {
		t.Fatalf("expected 403 for non-member, got %+v", apiErr)
	}
}

func TestOrgAccessResolvesMemberRole(t *testing.T) {
	app, org, user := setupAPIApp(t)

	e := newEvent(app, user, http.MethodGet, "/api/wa/inbox", "")
	e.Request.Header.Set("X-Org-Id", org.Id)

	access, apiErr := orgAccessFromRequest(e, app)
	if apiErr != nil {
		t.Fatalf("expected access for member, got %+v", apiErr)
	}
	if access.OrgID != org.Id || access.Role != store.RoleAdmin {
		t.Fatalf("unexpected access: %+v", access)
	}
}

func TestHandleLoginBadCredentials(t *testing.T) {
	app, _, _ := setupAPIApp(t)

	e := newEvent(app, nil, http.MethodPost, "/api/wa/auth/login",
		`{"email":"agent@example.com","password":"wrong"}`)

	err := HandleLogin(app)(e)
	var apiErr *router.ApiError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusUnauthorized {
		t.Fatalf("expected 401 ApiError, got %#v", err)
	}
}

func TestHandleLoginReturnsSession(t *testing.T) {
	app, org, _ := setupAPIApp(t)

	e := newEvent(app, nil, http.MethodPost, "/api/wa/auth/login",
		`{"email":"agent@example.com","password":"supersecret123"}`)

	if err := HandleLogin(app)(e); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	var session struct {
		Token string `json:"token"`
		User  struct {
			Email string `json:"email"`
		} `json:"user"`
		Orgs []struct {
			ID   string `json:"id"`
			Role string `json:"role"`
		} `json:"orgs"`
	}
	if err := json.Unmarshal(e.Response.(*httptest.ResponseRecorder).Body.Bytes(), &session); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	if session.Token == "" {
		t.Error("expected non-empty token")
	}
	if session.User.Email != "agent@example.com" {
		t.Errorf("user email = %q", session.User.Email)
	}
	if len(session.Orgs) != 1 || session.Orgs[0].ID != org.Id || session.Orgs[0].Role != store.RoleAdmin {
		t.Errorf("unexpected orgs in session: %+v", session.Orgs)
	}
}

func TestHandleBroadcastListIsOrgScopedWithBatchedNames(t *testing.T) {
	app, org, user := setupAPIApp(t)

	acc := saveAPIRecord(t, app, "whatsapp_accounts", map[string]any{
		"org": org.Id, "display_name": "Sales", "phone_number_id": "111", "access_token": "tok",
	})
	tmpl := saveAPIRecord(t, app, "message_templates", map[string]any{
		"org": org.Id, "account": acc.Id, "name": "hello", "language": "en_US",
		"body": "Hi", "status": "APPROVED",
	})
	bc := saveAPIRecord(t, app, "broadcasts", map[string]any{
		"org": org.Id, "account": acc.Id, "template": tmpl.Id,
		"name": "Launch", "status": store.BroadcastQueued,
	})

	// A broadcast in another org must not leak into the list.
	orgB := saveAPIRecord(t, app, "orgs", map[string]any{"name": "other"})
	saveAPIRecord(t, app, "broadcasts", map[string]any{
		"org": orgB.Id, "account": acc.Id, "template": tmpl.Id,
		"name": "Secret", "status": store.BroadcastQueued,
	})

	e := newEvent(app, user, http.MethodGet, "/api/wa/broadcasts", "")
	e.Request.Header.Set("X-Org-Id", org.Id)

	if err := HandleBroadcastList(app)(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	rec := e.Response.(*httptest.ResponseRecorder)
	var resp struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v (%s)", err, rec.Body.String())
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected exactly 1 org-scoped broadcast, got %d", len(resp.Items))
	}
	item := resp.Items[0]
	if item["id"] != bc.Id || item["name"] != "Launch" {
		t.Errorf("wrong broadcast surfaced: %v / %v", item["id"], item["name"])
	}
	if item["account_name"] != "Sales" {
		t.Errorf("account_name = %v (batched lookup failed), want Sales", item["account_name"])
	}
	if item["template_name"] != "hello" {
		t.Errorf("template_name = %v (batched lookup failed), want hello", item["template_name"])
	}
}
