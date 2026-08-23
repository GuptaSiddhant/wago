package queue

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/guptasiddhant/wago/pkg/meta"
	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// fakeSender records template sends and can be told to fail per phone number.
type fakeSender struct {
	mu      sync.Mutex
	calls   []string          // phones, in send order
	failFor map[string]bool   // phone -> always fail
	wamids  map[string]string // phone -> returned wamid
}

func newFakeSender() *fakeSender {
	return &fakeSender{failFor: map[string]bool{}, wamids: map[string]string{}}
}

func (f *fakeSender) SendText(ctx context.Context, accessToken, phoneNumberID, to, body string) (string, error) {
	return "", errors.New("not implemented in tests")
}

func (f *fakeSender) SendTemplate(ctx context.Context, accessToken, phoneNumberID, to, name, language string, params []map[string]any, header *meta.TemplateHeaderMedia) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, to)
	if f.failFor[to] {
		return "", errors.New("provider rejected " + to)
	}
	if f.wamids[to] == "" {
		f.wamids[to] = "wamid-" + strings.ToLower(to)
	}
	return f.wamids[to], nil
}

func (f *fakeSender) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// setupQueueApp builds a test app with collections and one broadcast holding
// numRecipients queued recipients.
func setupQueueApp(t *testing.T, numRecipients int) (*tests.TestApp, *core.Record, []*core.Record) {
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

	org := saveQueueRecord(t, app, "orgs", map[string]any{"name": "acme"})
	acc := saveQueueRecord(t, app, "whatsapp_accounts", map[string]any{
		"org": org.Id, "display_name": "Sales", "phone_number_id": "111",
		"access_token": "tok",
	})
	tmpl := saveQueueRecord(t, app, "message_templates", map[string]any{
		"org": org.Id, "account": acc.Id,
		"name": "hello_world", "language": "en_US", "body": "Hi", "status": "APPROVED",
	})
	bc := saveQueueRecord(t, app, "broadcasts", map[string]any{
		"org": org.Id, "account": acc.Id, "template": tmpl.Id,
		"name": "Launch", "status": store.BroadcastQueued,
	})

	recipients := make([]*core.Record, 0, numRecipients)
	for i := 0; i < numRecipients; i++ {
		c := saveQueueRecord(t, app, "contacts", map[string]any{
			"org": org.Id, "name": "C", "phone": "555000" + string(rune('1'+i)),
		})
		recipients = append(recipients, saveQueueRecord(t, app, "broadcast_recipients", map[string]any{
			"org": org.Id, "broadcast": bc.Id, "contact": c.Id,
			"phone": c.GetString("phone"), "status": store.RecipientQueued,
		}))
	}
	return app, bc, recipients
}

func saveQueueRecord(t *testing.T, app core.App, col string, vals map[string]any) *core.Record {
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

func reloadBroadcast(t *testing.T, app core.App, id string) *core.Record {
	t.Helper()
	bc, err := app.FindRecordById("broadcasts", id)
	if err != nil {
		t.Fatalf("reload broadcast: %v", err)
	}
	return bc
}

func recipientByPhone(t *testing.T, app core.App, bcID, phone string) *core.Record {
	t.Helper()
	rec, err := app.FindFirstRecordByFilter("broadcast_recipients",
		"broadcast = {:b} && phone = {:p}",
		store.DbxParams(map[string]any{"b": bcID, "p": phone}))
	if err != nil {
		t.Fatalf("recipient %s: %v", phone, err)
	}
	return rec
}

func TestProcessTickSendsAndFinalizes(t *testing.T) {
	app, bc, recipients := setupQueueApp(t, 3)
	sender := newFakeSender()
	worker := NewWorkerWithDeps(WorkerDeps{
		App:    app,
		Sender: sender,
		Config: Config{MessagesPerMinute: 6000}, // no rate-limit sleeps in tests
	})

	worker.ProcessTick(context.Background())

	if got := sender.callCount(); got != 3 {
		t.Fatalf("expected 3 sends, got %d", got)
	}

	sentCount := 0
	for _, r := range recipients {
		got := recipientByPhone(t, app, bc.Id, r.GetString("phone"))
		if got.GetString("status") != store.RecipientSent {
			t.Errorf("recipient %s status = %q, want sent (error=%q)",
				r.GetString("phone"), got.GetString("status"), got.GetString("error"))
			continue
		}
		if got.GetString("wamid") == "" {
			t.Errorf("recipient %s missing wamid", r.GetString("phone"))
		}
		sentCount++
	}
	if sentCount != 3 {
		t.Fatalf("expected 3 recipients sent, got %d", sentCount)
	}

	final := reloadBroadcast(t, app, bc.Id)
	if got := final.GetInt("sent_count"); got != 3 {
		t.Errorf("sent_count = %d, want 3", got)
	}
	if got := final.GetString("status"); got != store.BroadcastCompleted {
		t.Errorf("status = %q, want completed after drain", got)
	}
	if final.GetDateTime("finished_at").IsZero() {
		t.Error("expected finished_at set after finalize")
	}
}

func TestProcessTickRetriesThenFailsExhaustedRecipient(t *testing.T) {
	app, bc, recipients := setupQueueApp(t, 1)
	phone := recipients[0].GetString("phone")

	sender := newFakeSender()
	sender.failFor[phone] = true
	worker := NewWorkerWithDeps(WorkerDeps{
		App:    app,
		Sender: sender,
		Config: Config{MessagesPerMinute: 6000, MaxAttempts: 1},
	})

	// One tick exhausts the single allowed attempt -> final failure.
	worker.ProcessTick(context.Background())

	got := recipientByPhone(t, app, bc.Id, phone)
	if got.GetString("status") != store.RecipientFailed {
		t.Fatalf("status = %q, want failed", got.GetString("status"))
	}
	if got.GetInt("attempts") != 1 {
		t.Errorf("attempts = %d, want 1", got.GetInt("attempts"))
	}
	if !strings.Contains(got.GetString("error"), "failed after 1 attempts") {
		t.Errorf("error = %q, want exhausted-attempts message", got.GetString("error"))
	}

	final := reloadBroadcast(t, app, bc.Id)
	if got := final.GetInt("failed_count"); got != 1 {
		t.Errorf("failed_count = %d, want 1", got)
	}
	if got := final.GetString("status"); got != store.BroadcastFailed {
		t.Errorf("status = %q, want failed when everything failed", got)
	}
}

func TestProcessTickRetryKeepsRecipientQueuedWithinBudget(t *testing.T) {
	app, bc, recipients := setupQueueApp(t, 1)
	phone := recipients[0].GetString("phone")

	sender := newFakeSender()
	sender.failFor[phone] = true
	worker := NewWorkerWithDeps(WorkerDeps{
		App:    app,
		Sender: sender,
		Config: Config{MessagesPerMinute: 6000, MaxAttempts: 3},
	})

	worker.ProcessTick(context.Background())

	got := recipientByPhone(t, app, bc.Id, phone)
	if got.GetString("status") != store.RecipientQueued {
		t.Fatalf("status = %q, want requeued within retry budget", got.GetString("status"))
	}
	if got.GetInt("attempts") != 1 {
		t.Errorf("attempts = %d, want 1", got.GetInt("attempts"))
	}
	if got.GetDateTime("next_attempt_at").IsZero() {
		t.Error("expected backoff next_attempt_at set")
	}

	// With outstanding work remaining, the broadcast must stay running.
	final := reloadBroadcast(t, app, bc.Id)
	if got := final.GetString("status"); got != store.BroadcastRunning {
		t.Errorf("status = %q, want running while work is outstanding", got)
	}
}

func TestProcessTickSkipsCancelledBroadcast(t *testing.T) {
	app, bc, _ := setupQueueApp(t, 1)

	// Cancel before processing: the worker must not touch it.
	cancelled, _ := app.FindRecordById("broadcasts", bc.Id)
	cancelled.Set("status", store.BroadcastCancelled)
	if err := app.Save(cancelled); err != nil {
		t.Fatalf("cancel broadcast: %v", err)
	}

	sender := newFakeSender()
	worker := NewWorkerWithDeps(WorkerDeps{
		App:    app,
		Sender: sender,
		Config: Config{MessagesPerMinute: 6000},
	})

	worker.ProcessTick(context.Background())

	if sender.callCount() != 0 {
		t.Fatalf("cancelled broadcast was sent to, calls=%d", sender.callCount())
	}
}
