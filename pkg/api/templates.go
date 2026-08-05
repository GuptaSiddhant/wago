package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/guptasiddhant/wago/pkg/meta"
	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
)

// templateValidation errors returned before hitting Meta.
var (
	nameRe = regexp.MustCompile(`^[a-z0-9_]{1,512}$`)
)

type templateDTO struct {
	ID          string                `json:"id"`
	AccountID   string                `json:"account_id"`
	AccountName string                `json:"account_name"`
	MetaID      string                `json:"meta_id"`
	Name        string                `json:"name"`
	Language    string                `json:"language"`
	Category    string                `json:"category"`
	HeaderType  string                `json:"header_type"`
	HeaderText  string                `json:"header_text"`
	Body        string                `json:"body"`
	Footer      string                `json:"footer"`
	Buttons     []meta.TemplateButton `json:"buttons"`
	Status      string                `json:"status"`
	MetaError   string                `json:"meta_error"`
	Created     string                `json:"created"`
}

func templateFromRecord(app core.App, r *core.Record) templateDTO {
	dto := templateDTO{
		ID:         r.Id,
		AccountID:  r.GetString("account"),
		MetaID:     r.GetString("meta_id"),
		Name:       r.GetString("name"),
		Language:   r.GetString("language"),
		Category:   r.GetString("category"),
		HeaderType: r.GetString("header_type"),
		HeaderText: r.GetString("header_text"),
		Body:       r.GetString("body"),
		Footer:     r.GetString("footer"),
		Status:     r.GetString("status"),
		MetaError:  r.GetString("meta_error"),
		Buttons:    store.DecodeTemplateButtons(r),
		Created:    fmtDateTime(r.GetDateTime("created")),
	}
	if account, err := app.FindRecordById("whatsapp_accounts", dto.AccountID); err == nil {
		dto.AccountName = account.GetString("display_name")
	}
	return dto
}

// HandleTemplates lists the org's message templates, newest first.
func HandleTemplates(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		records, err := app.FindRecordsByFilter("message_templates", "org = {:org}", "-created", 500, 0,
			store.DbxParams(map[string]any{"org": access.OrgID}))
		if err != nil {
			return e.InternalServerError("Failed to list templates", err)
		}

		items := make([]templateDTO, 0, len(records))
		for _, r := range records {
			items = append(items, templateFromRecord(app, r))
		}
		return e.JSON(http.StatusOK, map[string]any{"items": items})
	}
}

type templateCreateRequest struct {
	AccountID     string                `json:"account_id" form:"account_id"`
	Name          string                `json:"name" form:"name"`
	Language      string                `json:"language" form:"language"`
	Category      string                `json:"category" form:"category"`
	HeaderType    string                `json:"header_type" form:"header_type"`
	HeaderText    string                `json:"header_text" form:"header_text"`
	Body          string                `json:"body" form:"body"`
	Footer        string                `json:"footer" form:"footer"`
	Buttons       []meta.TemplateButton `json:"buttons" form:"buttons"`
	ExampleValues []string              `json:"example_values" form:"example_values"`
}

// HandleTemplateCreate validates a submission, submits it to Meta for review,
// and stores the local record.
func HandleTemplateCreate(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		var body templateCreateRequest
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("Invalid request body", err)
		}

		body.Name = strings.TrimSpace(body.Name)
		body.Body = strings.TrimSpace(body.Body)
		body.HeaderType = strings.ToUpper(strings.TrimSpace(body.HeaderType))
		if body.HeaderType == "" {
			body.HeaderType = "NONE"
		}

		if body.AccountID == "" {
			return e.BadRequestError("account_id is required", nil)
		}
		if !nameRe.MatchString(body.Name) {
			return e.BadRequestError("template name must be lowercase letters, numbers and underscores", nil)
		}
		if body.Language == "" {
			return e.BadRequestError("language is required", nil)
		}
		switch body.Category {
		case "MARKETING", "UTILITY", "AUTHENTICATION":
		default:
			return e.BadRequestError("category must be MARKETING, UTILITY or AUTHENTICATION", nil)
		}
		if body.Body == "" {
			return e.BadRequestError("body text is required", nil)
		}
		if len(body.Body) > 1024 {
			return e.BadRequestError("body text must be <= 1024 characters", nil)
		}
		if body.HeaderType == "TEXT" && len(body.HeaderText) > 60 {
			return e.BadRequestError("header text must be <= 60 characters", nil)
		}
		if len(body.Footer) > 60 {
			return e.BadRequestError("footer text must be <= 60 characters", nil)
		}

		account, err := store.FindOrgRecord(app, access.OrgID, "whatsapp_accounts", body.AccountID)
		if err != nil {
			return e.BadRequestError("whatsapp account not found", nil)
		}
		token := account.GetString("access_token")
		wabaID := account.GetString("waba_id")
		if token == "" || wabaID == "" {
			return e.BadRequestError("this number needs an access token and WABA ID to create templates", nil)
		}

		submission := buildTemplateSubmission(body)
		client := meta.NewClient()
		metaID, err := client.CreateMessageTemplate(e.Request.Context(), token, wabaID, submission)
		if err != nil {
			return e.BadRequestError("Meta rejected the template: "+err.Error(), nil)
		}

		record, err := store.NewTemplateRecord(app, access.OrgID, body.AccountID)
		if err != nil {
			return e.InternalServerError("Failed to create template record", err)
		}
		record.Set("meta_id", metaID)
		record.Set("name", body.Name)
		record.Set("language", body.Language)
		record.Set("category", body.Category)
		record.Set("header_type", body.HeaderType)
		record.Set("header_text", body.HeaderText)
		record.Set("body", body.Body)
		record.Set("footer", body.Footer)
		record.Set("buttons", body.Buttons)
		record.Set("status", "PENDING")
		if err := app.Save(record); err != nil {
			return e.InternalServerError("Template created on Meta but failed to save locally", err)
		}

		return e.JSON(http.StatusCreated, templateFromRecord(app, record))
	}
}

// HandleTemplateDelete deletes a template locally and, when it exists on Meta,
// removes it from the WhatsApp Business Account first.
func HandleTemplateDelete(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		record, err := store.FindOrgTemplate(app, access.OrgID, e.Request.PathValue("id"))
		if err != nil {
			return e.NotFoundError("template not found", nil)
		}

		if metaID := record.GetString("meta_id"); metaID != "" {
			account, err := store.FindOrgRecord(app, access.OrgID, "whatsapp_accounts", record.GetString("account"))
			if err == nil {
				client := meta.NewClient()
				if err := client.DeleteMessageTemplate(e.Request.Context(),
					account.GetString("access_token"), account.GetString("waba_id"), metaID); err != nil {
					return e.BadRequestError("Failed to delete template on Meta: "+err.Error(), nil)
				}
			}
		}

		if err := app.Delete(record); err != nil {
			return e.InternalServerError("Failed to delete template", err)
		}
		return e.JSON(http.StatusOK, map[string]any{"id": record.Id})
	}
}

// HandleTemplatesSync pulls every template from the org's WABAs and upserts the
// local records so status changes (pending → approved/rejected) are reflected.
func HandleTemplatesSync(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		accounts, err := app.FindRecordsByFilter("whatsapp_accounts", "org = {:org}", "-created", 200, 0,
			store.DbxParams(map[string]any{"org": access.OrgID}))
		if err != nil {
			return e.InternalServerError("Failed to list accounts", err)
		}

		client := meta.NewClient()
		var errorsOut []string
		seen := map[string]bool{}

		for _, account := range accounts {
			token := account.GetString("access_token")
			wabaID := account.GetString("waba_id")
			if token == "" || wabaID == "" {
				continue
			}
			list, err := client.ListMessageTemplates(e.Request.Context(), token, wabaID)
			if err != nil {
				errorsOut = append(errorsOut, account.GetString("display_name")+": "+err.Error())
				continue
			}
			for i := range list {
				if _, err := store.UpsertTemplateFromMeta(app, access.OrgID, account.Id, &list[i]); err != nil {
					errorsOut = append(errorsOut, fmt.Sprintf("%s (%s): %v", list[i].Name, list[i].Status, err))
					continue
				}
				seen[list[i].ID] = true
			}
		}

		// Remove locally-synced templates that are no longer on Meta.
		records, err := app.FindRecordsByFilter("message_templates", "org = {:org} && meta_id != ''", "-created", 500, 0,
			store.DbxParams(map[string]any{"org": access.OrgID}))
		if err == nil {
			for _, r := range records {
				if !seen[r.GetString("meta_id")] {
					_ = app.Delete(r)
				}
			}
		}

		items, err := app.FindRecordsByFilter("message_templates", "org = {:org}", "-created", 500, 0,
			store.DbxParams(map[string]any{"org": access.OrgID}))
		if err != nil {
			return e.InternalServerError("Failed to list templates", err)
		}
		dtos := make([]templateDTO, 0, len(items))
		for _, r := range items {
			dtos = append(dtos, templateFromRecord(app, r))
		}

		return e.JSON(http.StatusOK, map[string]any{"items": dtos, "errors": errorsOut})
	}
}

// buildTemplateSubmission assembles the Meta Graph API components for a create.
func buildTemplateSubmission(body templateCreateRequest) *meta.TemplateSubmission {
	components := make([]meta.TemplateComponent, 0, 4)
	example := exampleBodyText(body.Body, body.ExampleValues)

	if body.HeaderType == "TEXT" && body.HeaderText != "" {
		components = append(components, meta.TemplateComponent{
			Type:    "HEADER",
			Text:    body.HeaderText,
			Example: map[string]any{"header_text": []string{body.HeaderText}},
		})
	}

	components = append(components, meta.TemplateComponent{
		Type:    "BODY",
		Text:    body.Body,
		Example: example,
	})

	if body.Footer != "" {
		components = append(components, meta.TemplateComponent{Type: "FOOTER", Text: body.Footer})
	}

	if len(body.Buttons) > 0 {
		components = append(components, meta.TemplateComponent{Type: "BUTTONS", Buttons: body.Buttons})
	}

	return &meta.TemplateSubmission{
		Name:       body.Name,
		Language:   body.Language,
		Category:   body.Category,
		Components: components,
	}
}

// exampleBodyText builds the body example payload. Meta expects a 2D array (one
// row per body message) of sample variable values; unused variables fall back
// to a friendly "Sample N" placeholder.
func exampleBodyText(body string, values []string) map[string]any {
	var vars []string
	if n := countVariables(body); n > 0 {
		for i := 0; i < n; i++ {
			sample := "Sample"
			if i < len(values) && strings.TrimSpace(values[i]) != "" {
				sample = values[i]
			}
			vars = append(vars, sample)
		}
	}
	if len(vars) == 0 {
		return nil
	}
	return map[string]any{"body_text": [][]string{vars}}
}

// countVariables returns the highest {{n}} index in the body text (0 if none).
func countVariables(body string) int {
	re := regexp.MustCompile(`\{\{(\d+)\}\}`)
	max := 0
	seen := false
	for _, m := range re.FindAllStringSubmatch(body, -1) {
		seen = true
		var n int
		if _, err := fmt.Sscanf(m[1], "%d", &n); err == nil && n > max {
			max = n
		}
	}
	if !seen {
		return 0
	}
	return max
}
