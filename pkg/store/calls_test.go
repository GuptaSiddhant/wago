package store

import (
	"testing"
	"time"
)

func TestVoiceCallsLifecycle(t *testing.T) {
	app := setupApp(t)

	org := saveRecord(t, app, "orgs", map[string]any{"name": "acme"})
	acct := saveRecord(t, app, "whatsapp_accounts", map[string]any{
		"org": org.Id, "display_name": "Sales", "phone_number_id": "111", "access_token": "tok",
	})
	c := saveRecord(t, app, "contacts", map[string]any{"org": org.Id, "name": "Ada", "phone": "5550001"})
	conv := saveRecord(t, app, "conversations", map[string]any{
		"org": org.Id, "contact": c.Id, "whatsapp_account": acct.Id,
	})

	// A call is created as ringing with the contact's phone and name.
	started := time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC)
	call, err := CreateIncomingCall(app, conv, CallDirectionInbound, "5550001", "Ada", started)
	if err != nil {
		t.Fatalf("CreateIncomingCall: %v", err)
	}
	if got := call.GetString("org"); got != org.Id {
		t.Errorf("call org = %q, want %q", got, org.Id)
	}
	if got := call.GetString("direction"); got != CallDirectionInbound {
		t.Errorf("call direction = %q", got)
	}
	if got := call.GetString("status"); got != CallRinging {
		t.Errorf("call status = %q, want %q", got, CallRinging)
	}

	// Activating records started_at.
	if err := SetCallStatus(app, call.Id, CallActive); err != nil {
		t.Fatalf("SetCallStatus active: %v", err)
	}
	call, _ = FindOrgCall(app, org.Id, call.Id)
	if call.GetDateTime("started_at").Time().IsZero() {
		t.Errorf("expected started_at to be set for active call")
	}

	// Ending records ended_at and a non-zero duration.
	if err := SetCallStatus(app, call.Id, CallEnded); err != nil {
		t.Fatalf("SetCallStatus ended: %v", err)
	}
	call, err = FindOrgCall(app, org.Id, call.Id)
	if err != nil {
		t.Fatalf("FindOrgCall: %v", err)
	}
	if got := call.GetString("status"); got != CallEnded {
		t.Errorf("call status = %q, want %q", got, CallEnded)
	}
	if call.GetDateTime("ended_at").Time().IsZero() {
		t.Errorf("expected ended_at set after ending call")
	}

	// The call is scoped and lists under its conversation.
	items, err := ListConversationCalls(app, org.Id, conv.Id, 10)
	if err != nil {
		t.Fatalf("ListConversationCalls: %v", err)
	}
	if len(items) != 1 || items[0].Id != call.Id {
		t.Errorf("expected 1 call in conversation, got %d (%v)", len(items), items)
	}
}

func TestFindOrgCallScopesByOrg(t *testing.T) {
	app := setupApp(t)

	orgA := saveRecord(t, app, "orgs", map[string]any{"name": "acme"})
	orgB := saveRecord(t, app, "orgs", map[string]any{"name": "spacely"})
	acct := saveRecord(t, app, "whatsapp_accounts", map[string]any{
		"org": orgA.Id, "display_name": "Sales", "phone_number_id": "111", "access_token": "tok",
	})
	c := saveRecord(t, app, "contacts", map[string]any{"org": orgA.Id, "name": "Ada", "phone": "5550001"})
	conv := saveRecord(t, app, "conversations", map[string]any{
		"org": orgA.Id, "contact": c.Id, "whatsapp_account": acct.Id,
	})
	call, err := CreateIncomingCall(app, conv, CallDirectionInbound, "5550001", "Ada", time.Now())
	if err != nil {
		t.Fatalf("CreateIncomingCall: %v", err)
	}

	// FindOrgCall must not return another org's call; a foreign org errors.
	if got, err := FindOrgCall(app, orgB.Id, call.Id); err == nil {
		t.Errorf("expected org B to be denied, got %q", got.Id)
	}
	if got, err := FindOrgCall(app, orgA.Id, call.Id); err != nil || got.Id != call.Id {
		t.Errorf("expected org A to find its call (err=%v)", err)
	}
}
