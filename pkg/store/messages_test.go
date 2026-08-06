package store

import (
	"testing"
)

func TestTemplateHeaderMedia(t *testing.T) {
	app := setupApp(t)

	org := saveRecord(t, app, "orgs", map[string]any{"name": "template media org"})
	acc := saveRecord(t, app, "whatsapp_accounts", map[string]any{
		"org":             org.Id,
		"display_name":    "Support",
		"phone_number_id": "12345",
		"access_token":    "tok",
	})

	// A template with a media header.
	mediaTpl := saveRecord(t, app, "message_templates", map[string]any{
		"org":               org.Id,
		"account":           acc.Id,
		"meta_id":           "meta_media",
		"name":              "welcome_with_image",
		"language":          "en_US",
		"category":          "UTILITY",
		"header_type":       "MEDIA",
		"header_media_type": "IMAGE",
		"header_media_id":   "himg_1",
		"header_media_name": "hero.png",
		"body":              "Welcome {{1}}",
		"status":            "APPROVED",
	})
	if h := TemplateHeaderMedia(mediaTpl); h == nil {
		t.Fatal("expected a header media override for a media template")
	} else if h.Kind != "image" || h.MediaID != "himg_1" {
		t.Errorf("unexpected header media: %+v", h)
	}

	// A template with a text header (no media) yields nil.
	textTpl := saveRecord(t, app, "message_templates", map[string]any{
		"org":         org.Id,
		"account":     acc.Id,
		"meta_id":     "meta_text",
		"name":        "text_only",
		"language":    "en_US",
		"category":    "UTILITY",
		"header_type": "TEXT",
		"header_text": "Hi",
		"body":        "Hello",
		"status":      "APPROVED",
	})
	if h := TemplateHeaderMedia(textTpl); h != nil {
		t.Errorf("expected no header media for a text template, got %+v", h)
	}

	// FindOrgTemplateByName resolves the media template by name.
	found, err := FindOrgTemplateByName(app, org.Id, "welcome_with_image")
	if err != nil {
		t.Fatalf("FindOrgTemplateByName: %v", err)
	}
	if found.Id != mediaTpl.Id {
		t.Errorf("FindOrgTemplateByName returned %s, want %s", found.Id, mediaTpl.Id)
	}
}

func TestEnsureMessagesCollectionHasMediaField(t *testing.T) {
	app := setupApp(t)

	col, err := app.FindCollectionByNameOrId("messages")
	if err != nil {
		t.Fatalf("messages collection not found: %v", err)
	}
	field := col.Fields.GetByName("media")
	if field == nil {
		t.Fatal("messages collection is missing a media file field")
	}
	fileField, ok := field.(interface{ IsMultiple() bool })
	if !ok {
		t.Fatalf("media field is not a file field (got %T)", field)
	}
	if fileField.IsMultiple() {
		t.Fatal("media field should hold a single file")
	}
}

func TestSaveOutgoingMessageStoresMedia(t *testing.T) {
	app := setupApp(t)

	org := saveRecord(t, app, "orgs", map[string]any{"name": "media org"})
	acc := saveRecord(t, app, "whatsapp_accounts", map[string]any{
		"org":             org.Id,
		"display_name":    "Support",
		"phone_number_id": "12345",
		"access_token":    "tok",
	})
	contact := saveRecord(t, app, "contacts", map[string]any{
		"org":   org.Id,
		"name":  "Jane",
		"phone": "15550001111",
	})
	conv := saveRecord(t, app, "conversations", map[string]any{
		"org":              org.Id,
		"contact":          contact.Id,
		"whatsapp_account": acc.Id,
	})

	msg, err := SaveOutgoingMessage(app, org.Id, conv.Id, "12345", "15550001111",
		"check this", "wamid-1", map[string]any{"type": "image"}, []byte("filedata"), "photo.jpg")
	if err != nil {
		t.Fatalf("SaveOutgoingMessage: %v", err)
	}

	if got := msg.GetString("media"); got == "" {
		t.Fatal("expected message to store a media file")
	}

	stored, err := FindMessageByWamid(app, org.Id, "wamid-1")
	if err != nil {
		t.Fatalf("FindMessageByWamid: %v", err)
	}
	if stored.GetString("wamid") != "wamid-1" {
		t.Errorf("wamid = %q, want wamid-1", stored.GetString("wamid"))
	}
}

func TestFindMessageByWamidIsOrgScoped(t *testing.T) {
	app := setupApp(t)

	orgA := saveRecord(t, app, "orgs", map[string]any{"name": "org a"})
	orgB := saveRecord(t, app, "orgs", map[string]any{"name": "org b"})
	accA := saveRecord(t, app, "whatsapp_accounts", map[string]any{
		"org": orgA.Id, "display_name": "A", "phone_number_id": "1", "access_token": "t",
	})
	convA := saveRecord(t, app, "conversations", map[string]any{
		"org": orgA.Id, "contact": saveRecord(t, app, "contacts", map[string]any{
			"org": orgA.Id, "name": "X", "phone": "1000",
		}).Id, "whatsapp_account": accA.Id,
	})

	if _, err := SaveOutgoingMessage(app, orgA.Id, convA.Id, "1", "1000",
		"hi", "wamid-scoped", map[string]any{"type": "text"}, nil, ""); err != nil {
		t.Fatalf("SaveOutgoingMessage: %v", err)
	}

	if _, err := FindMessageByWamid(app, orgB.Id, "wamid-scoped"); err == nil {
		t.Fatal("expected lookup from another org to fail")
	}

	if _, err := FindMessageByWamid(app, orgA.Id, "wamid-scoped"); err != nil {
		t.Fatalf("expected lookup from owning org to succeed: %v", err)
	}
}
