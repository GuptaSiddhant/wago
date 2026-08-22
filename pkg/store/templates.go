package store

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/guptasiddhant/wago/pkg/meta"

	"github.com/pocketbase/pocketbase/core"
)

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
	headerMediaType := ""
	var buttons []meta.TemplateButton
	for _, comp := range tmpl.Components {
		switch comp.Type {
		case "HEADER":
			if comp.Text != "" {
				headerType = "TEXT"
				headerText = comp.Text
			} else if comp.Format != "" {
				// Media header: report the format so the UI knows the template
				// carries media. The upload id is not recoverable from the API,
				// so it is left to local records/submissions.
				headerType = "MEDIA"
				headerMediaType = comp.Format
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
	record.Set("header_media_type", headerMediaType)
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

// FindOrgTemplateByName returns an org-scoped approved template by its name.
// It is used when sending templates that are referenced by name only.
func FindOrgTemplateByName(app core.App, orgID, name string) (*core.Record, error) {
	return app.FindFirstRecordByFilter("message_templates",
		"org = {:org} && name = {:name}",
		DbxParams(map[string]any{"org": orgID, "name": name}))
}

// TemplateHeaderMedia builds the send-time media override for a template's
// header from its stored media header fields, or nil when the template has no
// media header.
func TemplateHeaderMedia(r *core.Record) *meta.TemplateHeaderMedia {
	kind := strings.ToLower(r.GetString("header_media_type"))
	mediaID := r.GetString("header_media_id")
	if kind == "" || mediaID == "" {
		return nil
	}
	return &meta.TemplateHeaderMedia{Kind: kind, MediaID: mediaID}
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
