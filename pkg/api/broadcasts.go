package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
)

// HandleBroadcastList lists broadcasts for the org. Account names, template
// names and recipient progress are resolved with three batched queries instead
// of per-broadcast lookups (which made the list O(3n) queries).
func HandleBroadcastList(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		records, err := app.FindRecordsByFilter("broadcasts", "org = {:org}", "-created", 200, 0,
			store.DbxParams(map[string]any{"org": access.OrgID}))
		if err != nil {
			return e.InternalServerError("Failed to list broadcasts", err)
		}

		items := broadcastDTOList(app, records)
		return e.JSON(http.StatusOK, map[string]any{"items": items})
	}
}

// HandleBroadcastDetail returns a single broadcast with progress + recipients.
func HandleBroadcastDetail(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		bc, err := store.FindOrgBroadcast(app, access.OrgID, e.Request.PathValue("id"))
		if err != nil {
			return e.NotFoundError("broadcast not found", nil)
		}

		recipients, err := store.ListBroadcastRecipients(app, bc.Id, 100)
		if err != nil {
			return e.InternalServerError("Failed to load recipients", err)
		}
		recipientDTos := make([]map[string]any, 0, len(recipients))
		for _, r := range recipients {
			recipientDTos = append(recipientDTos, map[string]any{
				"id":        r.Id,
				"name":      r.GetString("name"),
				"phone":     r.GetString("phone"),
				"status":    r.GetString("status"),
				"wamid":     r.GetString("wamid"),
				"error":     r.GetString("error"),
				"available": r.GetDateTime("created").String(),
			})
		}

		return e.JSON(http.StatusOK, map[string]any{
			"broadcast":  broadcastFromRecord(app, bc),
			"recipients": recipientDTos,
		})
	}
}

// HandleBroadcastEvents streams live progress for a broadcast over SSE. The
// client connects with its auth headers (the fetch-based EventSource shim), so
// org scoping is enforced the same way as every other authed route.
func HandleBroadcastEvents(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		bc, err := store.FindOrgBroadcast(app, access.OrgID, e.Request.PathValue("id"))
		if err != nil {
			return e.NotFoundError("broadcast not found", nil)
		}
		broadcastID := bc.Id

		e.Response.Header().Set("Content-Type", "text/event-stream")
		e.Response.Header().Set("Cache-Control", "no-store")
		e.Response.Header().Set("X-Accel-Buffering", "no")

		// The stream stays open for the lifetime of the connection — lift the
		// server write deadline exactly like PocketBase's own realtime does.
		rc := http.NewResponseController(e.Response)
		rc.SetWriteDeadline(time.Time{})

		ctx := e.Request.Context()
		writeEvent := func(name string, dto broadcastDTO) error {
			payload, err := json.Marshal(dto)
			if err != nil {
				return err
			}
			if _, err := e.Response.Write([]byte("event: " + name + "\ndata: " + string(payload) + "\n\n")); err != nil {
				return err
			}
			return rc.Flush()
		}

		if err := writeEvent("snapshot", broadcastFromRecord(app, bc)); err != nil {
			return nil // client disconnected
		}

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				latest, err := store.FindOrgBroadcast(app, access.OrgID, broadcastID)
				if err != nil {
					return nil
				}
				dto := broadcastFromRecord(app, latest)
				if err := writeEvent("progress", dto); err != nil {
					return nil
				}
				switch dto.Status {
				case store.BroadcastCompleted, store.BroadcastFailed, store.BroadcastCancelled:
					_ = writeEvent("done", dto)
					return nil
				}
			}
		}
	}
}

type broadcastCreateRequest struct {
	Name            string           `json:"name" form:"name"`
	AccountID       string           `json:"account_id" form:"account_id"`
	TemplateID      string           `json:"template_id" form:"template_id"`
	Params          []map[string]any `json:"params" form:"params"`
	RatePerMinute   int              `json:"rate_per_minute" form:"rate_per_minute"`
	BatchSize       int              `json:"batch_size" form:"batch_size"`
	ContactIDs      []string         `json:"contact_ids" form:"contact_ids"`
	AllContacts     bool             `json:"all_contacts" form:"all_contacts"`
	HeaderMediaID   string           `json:"header_media_id" form:"header_media_id"`
	HeaderMediaType string           `json:"header_media_type" form:"header_media_type"`
	HeaderMediaName string           `json:"header_media_name" form:"header_media_name"`
}

// HandleBroadcastCreate creates a broadcast and its recipient queue. The worker
// picks it up within a few hundred milliseconds.
func HandleBroadcastCreate(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		var body broadcastCreateRequest
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("Invalid request body", err)
		}
		body.Name = strings.TrimSpace(body.Name)
		if body.AccountID == "" || body.TemplateID == "" {
			return e.BadRequestError("account_id and template_id are required", nil)
		}
		if body.Name == "" {
			return e.BadRequestError("name is required", nil)
		}
		if body.RatePerMinute < 1 {
			return e.BadRequestError("rate_per_minute must be >= 1", nil)
		}
		if body.BatchSize < 1 {
			return e.BadRequestError("batch_size must be >= 1", nil)
		}
		if !body.AllContacts && len(body.ContactIDs) == 0 {
			return e.BadRequestError("select recipients or choose all contacts", nil)
		}

		if _, err := store.FindOrgRecord(app, access.OrgID, "whatsapp_accounts", body.AccountID); err != nil {
			return e.BadRequestError("whatsapp account not found", nil)
		}
		tmpl, err := store.FindOrgTemplate(app, access.OrgID, body.TemplateID)
		if err != nil {
			return e.BadRequestError("template not found", nil)
		}
		if !strings.EqualFold(tmpl.GetString("status"), "APPROVED") {
			return e.BadRequestError("only approved templates can be broadcast", nil)
		}

		body.HeaderMediaType = strings.ToUpper(strings.TrimSpace(body.HeaderMediaType))
		if body.HeaderMediaID != "" {
			// A send-time media override is only valid for templates that
			// already carry a media header of the same kind.
			tmplHeaderType := strings.ToUpper(tmpl.GetString("header_media_type"))
			switch tmplHeaderType {
			case "IMAGE", "VIDEO", "DOCUMENT":
			default:
				return e.BadRequestError("the selected template has no media header; header media cannot be attached", nil)
			}
			if body.HeaderMediaType != tmplHeaderType {
				return e.BadRequestError("header_media_type must match the template header type ("+tmplHeaderType+")", nil)
			}
		} else {
			body.HeaderMediaType = ""
		}

		recipients, err := resolveBroadcastRecipients(app, access.OrgID, body)
		if err != nil {
			return e.BadRequestError(err.Error(), nil)
		}
		if len(recipients) == 0 {
			return e.BadRequestError("no contacts selected", nil)
		}

		createdBy := ""
		if e.Auth != nil {
			createdBy = e.Auth.Id
		}

		bc, err := store.CreateBroadcast(app, access.OrgID, body.AccountID, body.TemplateID,
			body.Name, createdBy, body.Params, body.RatePerMinute, body.BatchSize, recipients,
			body.HeaderMediaType, body.HeaderMediaID, body.HeaderMediaName)
		if err != nil {
			return e.InternalServerError("Failed to create broadcast", err)
		}

		return e.JSON(http.StatusCreated, broadcastFromRecord(app, bc))
	}
}

func resolveBroadcastRecipients(app core.App, orgID string, body broadcastCreateRequest) ([]store.RecipientSnapshot, error) {
	var contacts []*core.Record
	var err error

	if body.AllContacts {
		contacts, err = app.FindRecordsByFilter("contacts", "org = {:org}", "-created", 100000, 0,
			store.DbxParams(map[string]any{"org": orgID}))
		if err != nil {
			return nil, err
		}
	} else {
		for _, id := range body.ContactIDs {
			c, e := store.FindOrgRecord(app, orgID, "contacts", id)
			if e != nil {
				continue
			}
			contacts = append(contacts, c)
		}
	}

	snaps := make([]store.RecipientSnapshot, 0, len(contacts))
	for _, c := range contacts {
		phone := c.GetString("phone")
		if phone == "" {
			continue
		}
		snaps = append(snaps, store.RecipientSnapshot{
			ContactID: c.Id,
			Phone:     phone,
			Name:      c.GetString("name"),
		})
	}
	return snaps, nil
}

// HandleBroadcastCancel stops a queued/running broadcast.
func HandleBroadcastCancel(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		bc, err := store.FindOrgBroadcast(app, access.OrgID, e.Request.PathValue("id"))
		if err != nil {
			return e.NotFoundError("broadcast not found", nil)
		}
		switch bc.GetString("status") {
		case store.BroadcastQueued, store.BroadcastRunning:
		default:
			return e.BadRequestError("only queued or running broadcasts can be cancelled", nil)
		}
		bc.Set("status", store.BroadcastCancelled)
		bc.Set("finished_at", time.Now())
		if err := app.Save(bc); err != nil {
			return e.InternalServerError("Failed to cancel broadcast", err)
		}
		return e.JSON(http.StatusOK, broadcastFromRecord(app, bc))
	}
}

type broadcastDTO struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	AccountID       string `json:"account_id"`
	AccountName     string `json:"account_name"`
	TemplateID      string `json:"template_id"`
	TemplateName    string `json:"template_name"`
	HeaderMediaType string `json:"header_media_type,omitempty"`
	HeaderMediaID   string `json:"header_media_id,omitempty"`
	HeaderMediaName string `json:"header_media_name,omitempty"`
	RatePerMinute   int    `json:"rate_per_minute"`
	BatchSize       int    `json:"batch_size"`
	RecipientCount  int    `json:"recipient_count"`
	SentCount       int    `json:"sent_count"`
	FailedCount     int    `json:"failed_count"`
	Pending         int    `json:"pending"`
	Sending         int    `json:"sending"`
	Created         string `json:"created"`
	StartedAt       string `json:"started_at,omitempty"`
	FinishedAt      string `json:"finished_at,omitempty"`
}

func broadcastFromRecord(app core.App, r *core.Record) broadcastDTO {
	accountName := ""
	if account, err := app.FindRecordById("whatsapp_accounts", r.GetString("account")); err == nil {
		accountName = account.GetString("display_name")
	}
	templateName := ""
	if tmpl, err := app.FindRecordById("message_templates", r.GetString("template")); err == nil {
		templateName = tmpl.GetString("name")
	}
	var pending, sending int
	if counts, err := store.CountBroadcastRecipients(app, r.Id); err == nil {
		pending = counts[store.RecipientQueued] + counts[store.RecipientSending]
		sending = counts[store.RecipientSending]
	}
	return broadcastFromRecordParts(r, accountName, templateName, pending, sending)
}

// broadcastFromRecordParts assembles the DTO from pre-resolved values so list
// endpoints can batch their lookups instead of querying per record.
func broadcastFromRecordParts(r *core.Record, accountName, templateName string, pending, sending int) broadcastDTO {
	dto := broadcastDTO{
		ID:              r.Id,
		Name:            r.GetString("name"),
		Status:          r.GetString("status"),
		AccountID:       r.GetString("account"),
		AccountName:     accountName,
		TemplateID:      r.GetString("template"),
		TemplateName:    templateName,
		HeaderMediaType: r.GetString("header_media_type"),
		HeaderMediaID:   r.GetString("header_media_id"),
		HeaderMediaName: r.GetString("header_media_name"),
		RatePerMinute:   r.GetInt("rate_per_minute"),
		BatchSize:       r.GetInt("batch_size"),
		RecipientCount:  r.GetInt("recipient_count"),
		SentCount:       r.GetInt("sent_count"),
		FailedCount:     r.GetInt("failed_count"),
		Pending:         pending,
		Sending:         sending,
	}
	dto.Created = fmtDateTime(r.GetDateTime("created"))
	dto.StartedAt = fmtDateTime(r.GetDateTime("started_at"))
	dto.FinishedAt = fmtDateTime(r.GetDateTime("finished_at"))
	return dto
}

// broadcastDTOList builds DTOs for many broadcasts with a constant number of
// queries: one batched fetch per referenced collection plus one grouped count.
func broadcastDTOList(app core.App, records []*core.Record) []broadcastDTO {
	accountIDs := map[string]struct{}{}
	templateIDs := map[string]struct{}{}
	bcIDs := make([]string, 0, len(records))
	for _, r := range records {
		if id := r.GetString("account"); id != "" {
			accountIDs[id] = struct{}{}
		}
		if id := r.GetString("template"); id != "" {
			templateIDs[id] = struct{}{}
		}
		bcIDs = append(bcIDs, r.Id)
	}

	accountNames := batchNames(app, "whatsapp_accounts", "display_name", keys(accountIDs))
	templateNames := batchNames(app, "message_templates", "name", keys(templateIDs))
	counts, _ := store.CountBroadcastRecipientsBatch(app, bcIDs)

	items := make([]broadcastDTO, 0, len(records))
	for _, r := range records {
		var pending, sending int
		if c := counts[r.Id]; c != nil {
			pending = int(c[store.RecipientQueued] + c[store.RecipientSending])
			sending = int(c[store.RecipientSending])
		}
		items = append(items, broadcastFromRecordParts(r,
			accountNames[r.GetString("account")],
			templateNames[r.GetString("template")],
			pending, sending))
	}
	return items
}

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// batchNames resolves a display field for many record ids in one query.
func batchNames(app core.App, collection, field string, ids []string) map[string]string {
	names := make(map[string]string, len(ids))
	if len(ids) == 0 {
		return names
	}
	recs, err := app.FindRecordsByIds(collection, ids)
	if err != nil {
		return names // degrade to empty names rather than failing the list
	}
	for _, rec := range recs {
		names[rec.Id] = rec.GetString(field)
	}
	return names
}
