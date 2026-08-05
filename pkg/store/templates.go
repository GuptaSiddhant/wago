package store

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/guptasiddhant/wago/pkg/meta"

	"github.com/pocketbase/pocketbase/core"
)

// EnsureMessageTemplatesCollection creates the org-scoped message templates
// collection. Templates are cached locally (form + Meta review status) and
// synced with the WhatsApp Business Account.
func EnsureMessageTemplatesCollection(app core.App) error {
	orgsCol, err := app.FindCollectionByNameOrId("orgs")
	if err != nil {
		return fmt.Errorf("orgs collection not found: %w", err)
	}
	accountsCol, err := app.FindCollectionByNameOrId("whatsapp_accounts")
	if err != nil {
		return fmt.Errorf("whatsapp_accounts collection not found: %w", err)
	}

	col, err := app.FindCollectionByNameOrId("message_templates")
	if err != nil {
		col = core.NewBaseCollection("message_templates")

		// Templates are scoped to org_id and only reachable through the scoped
		// API (and PB superusers which always bypass rules).
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
				Name:          "account",
				CollectionId:  accountsCol.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.TextField{Name: "meta_id"},
			&core.TextField{Name: "name", Required: true},
			&core.TextField{Name: "language", Required: true},
			&core.SelectField{
				Name:      "category",
				MaxSelect: 1,
				Values:    []string{"MARKETING", "UTILITY", "AUTHENTICATION"},
			},
			&core.TextField{Name: "header_type"},
			&core.TextField{Name: "header_text"},
			&core.TextField{Name: "body", Required: true},
			&core.TextField{Name: "footer"},
			&core.JSONField{Name: "buttons"},
			&core.TextField{Name: "status", Required: true},
			&core.TextField{Name: "meta_error"},

			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		col.AddIndex("idx_msg_templates_meta_id", true, "meta_id", "")

		if err := app.Save(col); err != nil {
			return fmt.Errorf("failed to auto-create message_templates collection: %w", err)
		}
		log.Println("Auto-created 'message_templates' collection")
	} else {
		// Ensure fields exist even on pre-existing databases.
		ensureField(col, &core.TextField{Name: "meta_id"})
		ensureField(col, &core.TextField{Name: "header_type"})
		ensureField(col, &core.TextField{Name: "header_text"})
		ensureField(col, &core.TextField{Name: "footer"})
		ensureField(col, &core.JSONField{Name: "buttons"})
		ensureField(col, &core.TextField{Name: "meta_error"})
		if err := app.Save(col); err != nil {
			return err
		}
	}

	return nil
}

// UpsertTemplateFromMeta saves (or updates) a template as reported by Meta,
// keyed by its Meta id. Returns the local record.
func UpsertTemplateFromMeta(app core.App, orgID, accountID string, tmpl *meta.TemplateMeta) (*core.Record, error) {
	col, err := app.FindCollectionByNameOrId("message_templates")
	if err != nil {
		return nil, err
	}

	record, _ := app.FindFirstRecordByFilter("message_templates",
		"meta_id = {:meta_id} && org = {:org}",
		DbxParams(map[string]any{"meta_id": tmpl.ID, "org": orgID}))
	if record == nil {
		record = core.NewRecord(col)
		record.Set("org", orgID)
		record.Set("account", accountID)
		record.Set("meta_id", tmpl.ID)
	}

	record.Set("name", tmpl.Name)
	record.Set("language", tmpl.Language)
	record.Set("category", tmpl.Category)
	record.Set("status", tmpl.Status)
	record.Set("meta_error", tmpl.RejectedReason)

	// Reconstruct the editable fields from Meta's components so the local form
	// stays in sync with what is actually live.
	headerType, headerText, body, footer := "NONE", "", "", ""
	var buttons []meta.TemplateButton
	for _, comp := range tmpl.Components {
		switch comp.Type {
		case "HEADER":
			if comp.Text != "" {
				headerType = "TEXT"
				headerText = comp.Text
			}
		case "BODY":
			body = comp.Text
		case "FOOTER":
			footer = comp.Text
		case "BUTTONS":
			buttons = comp.Buttons
		}
	}
	record.Set("header_type", headerType)
	record.Set("header_text", headerText)
	record.Set("body", body)
	record.Set("footer", footer)
	if buttons == nil {
		buttons = []meta.TemplateButton{}
	}
	record.Set("buttons", buttons)

	if err := app.Save(record); err != nil {
		return nil, fmt.Errorf("failed to upsert template: %w", err)
	}
	return record, nil
}

// NewTemplateRecord creates a fresh local template record (before Meta submit).
func NewTemplateRecord(app core.App, orgID, accountID string) (*core.Record, error) {
	col, err := app.FindCollectionByNameOrId("message_templates")
	if err != nil {
		return nil, err
	}
	record := core.NewRecord(col)
	record.Set("org", orgID)
	record.Set("account", accountID)
	return record, nil
}

// FindOrgTemplate returns an org-scoped template by record id.
func FindOrgTemplate(app core.App, orgID, id string) (*core.Record, error) {
	return FindOrgRecord(app, orgID, "message_templates", id)
}

// DecodeTemplateButtons returns the JSON buttons field as a typed slice.
func DecodeTemplateButtons(record *core.Record) []meta.TemplateButton {
	var buttons []meta.TemplateButton
	if raw, ok := record.Get("buttons").(json.RawMessage); ok {
		_ = json.Unmarshal(raw, &buttons)
	}
	if buttons == nil {
		buttons = []meta.TemplateButton{}
	}
	return buttons
}
