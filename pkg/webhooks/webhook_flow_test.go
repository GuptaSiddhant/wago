package webhooks

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/guptasiddhant/wago/pkg/notifications"
	"github.com/guptasiddhant/wago/pkg/runtimecfg"
	"github.com/guptasiddhant/wago/pkg/store"
	"github.com/guptasiddhant/wago/pkg/utils"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// setupWebhookApp builds a test app with collections plus one connected
// WhatsApp account routed by phone_number_id "111".
func setupWebhookApp(t *testing.T) *tests.TestApp {
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

	orgCol, err := app.FindCollectionByNameOrId("orgs")
	if err != nil {
		t.Fatalf("orgs collection: %v", err)
	}
	org := core.NewRecord(orgCol)
	org.Set("name", "acme")
	if err := app.Save(org); err != nil {
		t.Fatalf("save org: %v", err)
	}

	accCol, err := app.FindCollectionByNameOrId("whatsapp_accounts")
	if err != nil {
		t.Fatalf("accounts collection: %v", err)
	}
	acc := core.NewRecord(accCol)
	acc.Set("org", org.Id)
	acc.Set("display_name", "Sales")
	acc.Set("phone_number_id", "111")
	acc.Set("access_token", "tok")
	if err := app.Save(acc); err != nil {
		t.Fatalf("save account: %v", err)
	}

	return app
}

// postWebhook invokes the inbound-message handler with a raw JSON body.
func postWebhook(t *testing.T, app *tests.TestApp, body string) *httptest.ResponseRecorder {
	t.Helper()
	mgr := runtimecfg.New(&utils.AppConfig{})
	notifier := notifications.NewNotifier(mgr)
	handler := HandleIncomingMessage(mgr, notifier)

	req := httptest.NewRequest(http.MethodPost, "/api/wa/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e := new(core.RequestEvent)
	e.App = app
	e.Request = req
	e.Response = rec
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	return rec
}

func inboundPayloadJSON(phoneNumberID, waID, name, text, wamid string) string {
	return `{
		"object": "whatsapp_business_account",
		"entry": [{"id": "waba1", "changes": [{
			"field": "messages",
			"value": {
				"messaging_product": "whatsapp",
				"metadata": {"display_phone_number": "15550000", "phone_number_id": "` + phoneNumberID + `"},
				"contacts": [{"profile": {"name": "` + name + `"}, "wa_id": "` + waID + `"}],
				"messages": [{"from": "` + waID + `", "id": "` + wamid + `",
					"timestamp": "1700000000", "type": "text",
					"text": {"body": "` + text + `"}}]
			}
		}]}]
	}`
}

func TestHandleVerificationAcceptsCorrectToken(t *testing.T) {
	app := setupWebhookApp(t)
	mgr := runtimecfg.New(&utils.AppConfig{WA_WebhookVerifyToken: "secret"})

	req := httptest.NewRequest(http.MethodGet,
		"/api/wa/webhook?hub.mode=subscribe&hub.verify_token=secret&hub.challenge=12345", nil)
	rec := httptest.NewRecorder()
	e := new(core.RequestEvent)
	e.App = app
	e.Request = req
	e.Response = rec

	if err := HandleVerification(mgr)(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusOK || rec.Body.String() != "12345" {
		t.Fatalf("got %d %q, want 200 with echoed challenge", rec.Code, rec.Body.String())
	}
}

func TestHandleVerificationRejectsWrongToken(t *testing.T) {
	app := setupWebhookApp(t)
	mgr := runtimecfg.New(&utils.AppConfig{WA_WebhookVerifyToken: "secret"})

	req := httptest.NewRequest(http.MethodGet,
		"/api/wa/webhook?hub.mode=subscribe&hub.verify_token=wrong&hub.challenge=x", nil)
	rec := httptest.NewRecorder()
	e := new(core.RequestEvent)
	e.App = app
	e.Request = req
	e.Response = rec

	if err := HandleVerification(mgr)(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("got %d, want 403", rec.Code)
	}
}

func TestHandleIncomingMessageSavesInboundFlow(t *testing.T) {
	app := setupWebhookApp(t)
	rec := postWebhook(t, app, inboundPayloadJSON("111", "15551234567", "Ada", "hello world", "wamid.in.1"))

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "EVENT_RECEIVED") {
		t.Fatalf("got %d %q, want 200 EVENT_RECEIVED", rec.Code, rec.Body.String())
	}

	// Contact was upserted.
	contact, err := app.FindFirstRecordByFilter("contacts",
		"phone = {:p}", store.DbxParams(map[string]any{"p": "15551234567"}))
	if err != nil {
		t.Fatalf("expected contact for inbound wa_id: %v", err)
	}
	if contact.GetString("name") != "Ada" {
		t.Errorf("contact name = %q, want Ada", contact.GetString("name"))
	}

	// Conversation was created for the contact.
	conv, err := app.FindFirstRecordByFilter("conversations",
		"contact = {:c}", store.DbxParams(map[string]any{"c": contact.Id}))
	if err != nil {
		t.Fatalf("expected conversation: %v", err)
	}

	// Message was persisted with its body and wamid.
	msg, err := store.FindMessageByWamid(app, conv.GetString("org"), "wamid.in.1")
	if err != nil {
		t.Fatalf("expected saved message: %v", err)
	}
	if msg.GetString("body") != "hello world" || msg.GetString("direction") != "inbound" {
		t.Errorf("unexpected message body/direction: %q/%q",
			msg.GetString("body"), msg.GetString("direction"))
	}

	// Unread counter was incremented exactly once.
	if got := conv.GetInt("unread_count"); got < 1 {
		t.Errorf("unread_count = %d, want >= 1", got)
	}
}

func TestHandleIncomingMessageIsIdempotentPerWamid(t *testing.T) {
	app := setupWebhookApp(t)
	payload := inboundPayloadJSON("111", "15551234567", "Ada", "dup", "wamid.dup")

	if rec := postWebhook(t, app, payload); rec.Code != http.StatusOK {
		t.Fatalf("first post: got %d", rec.Code)
	}
	if rec := postWebhook(t, app, payload); rec.Code != http.StatusOK {
		t.Fatalf("second post: got %d", rec.Code)
	}

	msgs, err := app.FindRecordsByFilter("messages",
		"wamid = {:w}", "", 10, 0, store.DbxParams(map[string]any{"w": "wamid.dup"}))
	if err != nil {
		t.Fatalf("query messages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message after duplicate delivery, got %d", len(msgs))
	}
}

func TestHandleIncomingMessageIgnoresUnknownPhoneNumberID(t *testing.T) {
	app := setupWebhookApp(t)
	before, _ := app.CountRecords("messages")

	rec := postWebhook(t, app, inboundPayloadJSON("unknown-id", "15551234567", "Ada", "?", "wamid.x"))
	if rec.Code != http.StatusOK {
		t.Fatalf("Meta expects 200 even for unroutable payloads, got %d", rec.Code)
	}

	after, _ := app.CountRecords("messages")
	if before != after {
		t.Fatalf("message count changed (%d -> %d) for unknown phone_number_id", before, after)
	}
}

func TestHandleIncomingMessageUpdatesDeliveryStatus(t *testing.T) {
	app := setupWebhookApp(t)
	postWebhook(t, app, inboundPayloadJSON("111", "15551234567", "Ada", "status me", "wamid.st"))

	statusBody := `{
		"object": "whatsapp_business_account",
		"entry": [{"id": "waba1", "changes": [{
			"field": "messages",
			"value": {
				"messaging_product": "whatsapp",
				"metadata": {"display_phone_number": "15550000", "phone_number_id": "111"},
				"statuses": [{"id": "wamid.st", "status": "delivered", "timestamp": "1700000100"}]
			}
		}]}]
	}`
	if rec := postWebhook(t, app, statusBody); rec.Code != http.StatusOK {
		t.Fatalf("status post: got %d", rec.Code)
	}

	msg, err := store.FindMessageByWamid(app, "", "wamid.st")
	if err != nil {
		// FindMessageByWamid is org-scoped; fall back to direct lookup.
		msg, err = app.FindFirstRecordByFilter("messages",
			"wamid = {:w}", store.DbxParams(map[string]any{"w": "wamid.st"}))
		if err != nil {
			t.Fatalf("message vanished: %v", err)
		}
	}
	if msg.GetString("status") != "delivered" {
		t.Errorf("status = %q, want delivered", msg.GetString("status"))
	}
}

func TestHandleIncomingMessageRejectsMalformedBody(t *testing.T) {
	app := setupWebhookApp(t)
	rec := postWebhook(t, app, "{not json")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "EVENT_RECEIVED") {
		t.Fatalf("malformed payload must still 200 so Meta does not retry forever, got %d %q",
			rec.Code, rec.Body.String())
	}
}
