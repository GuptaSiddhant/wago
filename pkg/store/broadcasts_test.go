package store

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// setupApp builds a fresh PocketBase test app with all Wago collections.
func setupApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	// The pocketbase test data dir ships without a users auth collection; the
	// broadcasts schema references it, so create one before ensuring collections.
	if _, err := app.FindCollectionByNameOrId("users"); err != nil {
		if err := app.Save(core.NewAuthCollection("users")); err != nil {
			t.Fatalf("failed to create users collection: %v", err)
		}
	}

	if err := EnsureCollections(app); err != nil {
		t.Fatalf("failed to ensure collections: %v", err)
	}
	return app
}

// saveRecord is a tiny helper to persist a record in a collection.
func saveRecord(t *testing.T, app core.App, colName string, vals map[string]any) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId(colName)
	if err != nil {
		t.Fatalf("find collection %s: %v", colName, err)
	}
	rec := core.NewRecord(col)
	for k, v := range vals {
		rec.Set(k, v)
	}
	if err := app.Save(rec); err != nil {
		t.Fatalf("save %s: %v", colName, err)
	}
	return rec
}

func TestClaimDueRecipients(t *testing.T) {
	app := setupApp(t)

	org := saveRecord(t, app, "orgs", map[string]any{"name": "acme"})
	acct := saveRecord(t, app, "whatsapp_accounts", map[string]any{
		"org": org.Id, "display_name": "Sales", "phone_number_id": "111", "access_token": "tok",
	})
	tmpl := saveRecord(t, app, "message_templates", map[string]any{
		"org": org.Id, "account": acct.Id, "name": "hello_world", "language": "en_US",
		"status": "APPROVED", "body": "Hello {{1}}",
	})
	c1 := saveRecord(t, app, "contacts", map[string]any{"org": org.Id, "name": "A", "phone": "5550001"})
	c2 := saveRecord(t, app, "contacts", map[string]any{"org": org.Id, "name": "B", "phone": "5550002"})

	bc, err := CreateBroadcast(app, org.Id, acct.Id, tmpl.Id, "test", "", nil, 60, 10, []RecipientSnapshot{
		{ContactID: c1.Id, Phone: "5550001", Name: "A"},
		{ContactID: c2.Id, Phone: "5550002", Name: "B"},
	}, "IMAGE", "mid_1", "hero.png")
	if err != nil {
		t.Fatalf("CreateBroadcast: %v", err)
	}
	if bc.GetString("header_media_type") != "IMAGE" || bc.GetString("header_media_id") != "mid_1" {
		t.Errorf("broadcast header media not stored: type=%q id=%q",
			bc.GetString("header_media_type"), bc.GetString("header_media_id"))
	}

	claimed, err := ClaimDueRecipients(app, bc.Id, 10, 300)
	if err != nil {
		t.Fatalf("ClaimDueRecipients: %v", err)
	}
	if len(claimed) != 2 {
		t.Fatalf("expected 2 claimed, got %d", len(claimed))
	}
	for _, r := range claimed {
		if r.GetString("status") != RecipientSending {
			t.Errorf("expected status sending, got %q", r.GetString("status"))
		}
		if r.GetInt("attempts") != 1 {
			t.Errorf("expected attempts 1, got %d", r.GetInt("attempts"))
		}
		if r.GetDateTime("lease_until").IsZero() {
			t.Error("expected lease_until to be set")
		}
	}

	again, err := ClaimDueRecipients(app, bc.Id, 10, 300)
	if err != nil {
		t.Fatalf("second ClaimDueRecipients: %v", err)
	}
	if len(again) != 0 {
		t.Errorf("expected no double claim, got %d", len(again))
	}
}

func TestClaimRespectsBackoffWindow(t *testing.T) {
	app := setupApp(t)

	org := saveRecord(t, app, "orgs", map[string]any{"name": "acme"})
	acct := saveRecord(t, app, "whatsapp_accounts", map[string]any{
		"org": org.Id, "phone_number_id": "111", "access_token": "tok",
	})
	tmpl := saveRecord(t, app, "message_templates", map[string]any{
		"org": org.Id, "account": acct.Id, "name": "hello_world", "language": "en_US",
		"status": "APPROVED", "body": "Hi",
	})
	c1 := saveRecord(t, app, "contacts", map[string]any{"org": org.Id, "name": "A", "phone": "5550001"})

	bc, err := CreateBroadcast(app, org.Id, acct.Id, tmpl.Id, "test", "", nil, 60, 10,
		[]RecipientSnapshot{{ContactID: c1.Id, Phone: "5550001", Name: "A"}}, "", "", "")
	if err != nil {
		t.Fatalf("CreateBroadcast: %v", err)
	}

	claimed, err := ClaimDueRecipients(app, bc.Id, 10, 300)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim: %v (got %d)", err, len(claimed))
	}
	rec := claimed[0]

	// First attempt fails but stays under maxAttempts -> requeued with backoff.
	failed, err := RetryOrFailRecipient(app, rec, 3, time.Hour)
	if err != nil {
		t.Fatalf("RetryOrFailRecipient: %v", err)
	}
	if failed {
		t.Fatal("expected recipient to be retried, not failed")
	}
	if rec.GetString("status") != RecipientQueued {
		t.Errorf("expected requeued status, got %q", rec.GetString("status"))
	}

	// The future next_attempt_at must exclude it from the next claim.
	again, err := ClaimDueRecipients(app, bc.Id, 10, 300)
	if err != nil {
		t.Fatalf("claim after retry: %v", err)
	}
	if len(again) != 0 {
		t.Errorf("expected recipient excluded while backing off, got %d claims", len(again))
	}

	// Backoff elapses -> eligible again.
	if _, err := app.DB().NewQuery(
		"UPDATE broadcast_recipients SET next_attempt_at = {:past}").
		Bind(DbxParams(map[string]any{"past": sqlTime(time.Now().Add(-time.Minute))})).
		Execute(); err != nil {
		t.Fatalf("fast-forward backoff: %v", err)
	}
	due, err := ClaimDueRecipients(app, bc.Id, 10, 300)
	if err != nil || len(due) != 1 {
		t.Fatalf("claim after backoff: %v (got %d)", err, len(due))
	}
	if due[0].GetInt("attempts") != 2 {
		t.Errorf("expected attempts 2, got %d", due[0].GetInt("attempts"))
	}
}

func TestRetryExhaustsToFailed(t *testing.T) {
	app := setupApp(t)

	org := saveRecord(t, app, "orgs", map[string]any{"name": "acme"})
	acct := saveRecord(t, app, "whatsapp_accounts", map[string]any{
		"org": org.Id, "phone_number_id": "111", "access_token": "tok",
	})
	tmpl := saveRecord(t, app, "message_templates", map[string]any{
		"org": org.Id, "account": acct.Id, "name": "hello_world", "language": "en_US",
		"status": "APPROVED", "body": "Hi",
	})
	c1 := saveRecord(t, app, "contacts", map[string]any{"org": org.Id, "name": "A", "phone": "5550001"})

	bc, err := CreateBroadcast(app, org.Id, acct.Id, tmpl.Id, "test", "", nil, 60, 10,
		[]RecipientSnapshot{{ContactID: c1.Id, Phone: "5550001", Name: "A"}}, "", "", "")
	if err != nil {
		t.Fatalf("CreateBroadcast: %v", err)
	}

	claimed, err := ClaimDueRecipients(app, bc.Id, 10, 300)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim: %v (got %d)", err, len(claimed))
	}

	// maxAttempts=1 -> the current attempt (1) is already the last one.
	failed, err := RetryOrFailRecipient(app, claimed[0], 1, time.Second)
	if err != nil {
		t.Fatalf("RetryOrFailRecipient: %v", err)
	}
	if !failed {
		t.Fatal("expected recipient to be marked failed")
	}
	if claimed[0].GetString("status") != RecipientFailed {
		t.Errorf("expected failed status, got %q", claimed[0].GetString("status"))
	}
}

func TestReleaseExpiredLeases(t *testing.T) {
	app := setupApp(t)

	org := saveRecord(t, app, "orgs", map[string]any{"name": "acme"})
	acct := saveRecord(t, app, "whatsapp_accounts", map[string]any{
		"org": org.Id, "phone_number_id": "111", "access_token": "tok",
	})
	tmpl := saveRecord(t, app, "message_templates", map[string]any{
		"org": org.Id, "account": acct.Id, "name": "hello_world", "language": "en_US",
		"status": "APPROVED", "body": "Hi",
	})
	c1 := saveRecord(t, app, "contacts", map[string]any{"org": org.Id, "name": "A", "phone": "5550001"})

	bc, err := CreateBroadcast(app, org.Id, acct.Id, tmpl.Id, "test", "", nil, 60, 10,
		[]RecipientSnapshot{{ContactID: c1.Id, Phone: "5550001", Name: "A"}}, "", "", "")
	if err != nil {
		t.Fatalf("CreateBroadcast: %v", err)
	}

	claimed, err := ClaimDueRecipients(app, bc.Id, 10, 300)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim: %v (got %d)", err, len(claimed))
	}

	// Nothing should be released while the lease is still live.
	released, err := ReleaseExpiredLeases(app, time.Now())
	if err != nil {
		t.Fatalf("ReleaseExpiredLeases: %v", err)
	}
	if released != 0 {
		t.Errorf("expected 0 releases with a live lease, got %d", released)
	}

	// Force the lease into the past (simulates a crashed worker) and sweep.
	if _, err := app.DB().NewQuery(
		"UPDATE broadcast_recipients SET lease_until = {:past}").
		Bind(DbxParams(map[string]any{"past": sqlTime(time.Now().Add(-time.Hour))})).
		Execute(); err != nil {
		t.Fatalf("expire leases: %v", err)
	}
	released, err = ReleaseExpiredLeases(app, time.Now())
	if err != nil {
		t.Fatalf("ReleaseExpiredLeases: %v", err)
	}
	if released != 1 {
		t.Errorf("expected 1 release, got %d", released)
	}

	// The recipient must be claimable again now.
	again, err := ClaimDueRecipients(app, bc.Id, 10, 300)
	if err != nil || len(again) != 1 {
		t.Fatalf("re-claim after release: %v (got %d)", err, len(again))
	}
	if again[0].GetInt("attempts") != 2 {
		t.Errorf("expected attempts 2 after redelivery, got %d", again[0].GetInt("attempts"))
	}
}
