package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/pocketbase/dbx"

	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
)

// WhatsApp account connection statuses.
const (
	accountStatusConnected    = "connected"
	accountStatusDisconnected = "disconnected"
)

// waAccountDTO is a WhatsApp account listing for the settings page.
type waAccountDTO struct {
	ID            string `json:"id"`
	DisplayName   string `json:"display_name"`
	PhoneNumberID string `json:"phone_number_id"`
	WabaID        string `json:"waba_id,omitempty"`
	Status        string `json:"status"`
	TeamID        string `json:"team_id,omitempty"`
	TeamName      string `json:"team_name,omitempty"`
}

// teamMemberDTO is an org member listing for the settings page.
type teamMemberDTO struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	TeamID   string `json:"team_id,omitempty"`
	TeamName string `json:"team_name,omitempty"`
}

// HandleContacts lists contacts for the current org.
func HandleContacts(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		q := e.Request.URL.Query()
		search := strings.TrimSpace(q.Get("search"))

		limit, _ := strconv.Atoi(q.Get("limit"))
		if limit <= 0 || limit > 200 {
			limit = 50
		}
		offset, _ := strconv.Atoi(q.Get("offset"))
		if offset < 0 {
			offset = 0
		}

		params := map[string]any{"org": access.OrgID}
		filter := "org = {:org}"
		if search != "" {
			params["q"] = search
			filter += " && (name ~ {:q} || phone ~ {:q})"
		}

		records, err := app.FindRecordsByFilter("contacts", filter, "-created", limit, offset,
			store.DbxParams(params))
		if err != nil {
			return e.InternalServerError("Failed to list contacts", err)
		}

		items := make([]contactDTO, 0, len(records))
		for _, r := range records {
			items = append(items, contactToDTO(r))
		}

		return e.JSON(http.StatusOK, map[string]any{"items": items})
	}
}

// HandleTeam lists the members of the current org.
func HandleTeam(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		records, err := app.FindRecordsByFilter("org_members", "org = {:org}", "-created", 200, 0,
			store.DbxParams(map[string]any{"org": access.OrgID}))
		if err != nil {
			return e.InternalServerError("Failed to list team members", err)
		}

		items := make([]teamMemberDTO, 0, len(records))
		for _, m := range records {
			user, err := app.FindRecordById("users", m.GetString("user"))
			if err != nil {
				continue
			}
			items = append(items, teamMemberFromRecords(app, user, m))
		}

		return e.JSON(http.StatusOK, map[string]any{"items": items})
	}
}

// HandleAccounts lists the WhatsApp numbers connected to the current org.
func HandleAccounts(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		records, err := app.FindRecordsByFilter("whatsapp_accounts", "org = {:org}", "-created", 200, 0,
			store.DbxParams(map[string]any{"org": access.OrgID}))
		if err != nil {
			return e.InternalServerError("Failed to list WhatsApp accounts", err)
		}

		items := make([]waAccountDTO, 0, len(records))
		for _, r := range records {
			items = append(items, waAccountFromRecord(app, r))
		}

		return e.JSON(http.StatusOK, map[string]any{"items": items})
	}
}

func waAccountFromRecord(app core.App, r *core.Record) waAccountDTO {
	dto := waAccountDTO{
		ID:            r.Id,
		DisplayName:   r.GetString("display_name"),
		PhoneNumberID: r.GetString("phone_number_id"),
		WabaID:        r.GetString("waba_id"),
		Status:        r.GetString("status"),
	}
	if teamID := r.GetString("team"); teamID != "" {
		dto.TeamID = teamID
		if team, err := app.FindRecordById("teams", teamID); err == nil {
			dto.TeamName = team.GetString("name")
		}
	}
	return dto
}

// countConversationsFor counts org-scoped rows referencing a record (used to
// refuse deletes of contacts/accounts that still have conversation history).
// Uses an aggregate COUNT so large histories can't slip through a row cap.
func countConversationsFor(app core.App, collectionName, refField, orgID, refID string) (int, error) {
	count, err := app.CountRecords(collectionName,
		dbx.And(dbx.HashExp{"org": orgID, refField: refID}))
	return int(count), err
}

// HandleContactCreate creates a contact in the current org.
func HandleContactCreate(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		if !access.CanManageData() {
			return e.ForbiddenError("only the owner or a superadmin can manage contacts", nil)
		}

		var body struct {
			Name  string   `json:"name" form:"name"`
			Phone string   `json:"phone" form:"phone"`
			Tags  []string `json:"tags" form:"tags"`
			Notes *string  `json:"notes" form:"notes"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("Invalid request body", err)
		}
		body.Phone = strings.TrimSpace(body.Phone)
		body.Name = strings.TrimSpace(body.Name)
		if body.Phone == "" {
			return e.BadRequestError("phone is required", nil)
		}

		if existing, err := store.FindContactByPhone(app, access.OrgID, body.Phone); err == nil {
			return e.JSON(http.StatusOK, contactToDTO(existing))
		}

		contact, err := store.UpsertContact(app, access.OrgID, body.Phone, body.Name)
		if err != nil {
			return e.InternalServerError("failed to create contact", err)
		}
		if body.Tags != nil {
			contact.Set("tags", body.Tags)
		}
		if body.Notes != nil {
			contact.Set("notes", *body.Notes)
		}
		if body.Tags != nil || body.Notes != nil {
			if err := app.Save(contact); err != nil {
				return e.InternalServerError("failed to save contact", err)
			}
		}
		return e.JSON(http.StatusOK, contactToDTO(contact))
	}
}

// HandleContactUpdate updates a contact in the current org. Name and phone are
// restricted to owners/superadmins; agents may edit tags and notes.
func HandleContactUpdate(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		contact, err := store.FindOrgRecord(app, access.OrgID, "contacts", e.Request.PathValue("id"))
		if err != nil {
			return e.NotFoundError("contact not found", nil)
		}

		var body struct {
			Name  string   `json:"name" form:"name"`
			Phone string   `json:"phone" form:"phone"`
			Tags  []string `json:"tags" form:"tags"`
			Notes *string  `json:"notes" form:"notes"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("Invalid request body", err)
		}

		name, phone := strings.TrimSpace(body.Name), strings.TrimSpace(body.Phone)
		if name == "" && phone == "" && body.Tags == nil && body.Notes == nil {
			return e.BadRequestError("nothing to update", nil)
		}
		if (name != "" || phone != "") && !access.CanManageData() {
			return e.ForbiddenError("only the owner or a superadmin can edit contact name or phone", nil)
		}
		if (body.Tags != nil || body.Notes != nil) && !access.CanAssign() {
			return e.ForbiddenError("your role cannot edit contact details", nil)
		}

		if phone != "" {
			contact.Set("phone", phone)
		}
		if name != "" {
			contact.Set("name", name)
		}
		if body.Tags != nil {
			contact.Set("tags", body.Tags)
		}
		if body.Notes != nil {
			contact.Set("notes", *body.Notes)
		}
		if err := app.Save(contact); err != nil {
			return e.InternalServerError("failed to update contact", err)
		}
		return e.JSON(http.StatusOK, contactToDTO(contact))
	}
}

// HandleContactDelete deletes a contact that has no conversation history.
func HandleContactDelete(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		if !access.CanManageData() {
			return e.ForbiddenError("only the owner or a superadmin can manage contacts", nil)
		}

		contact, err := store.FindOrgRecord(app, access.OrgID, "contacts", e.Request.PathValue("id"))
		if err != nil {
			return e.NotFoundError("contact not found", nil)
		}

		count, err := countConversationsFor(app, "conversations", "contact", access.OrgID, contact.Id)
		if err != nil {
			return e.InternalServerError("Failed to check conversations", err)
		}
		if count > 0 {
			return e.BadRequestError("cannot delete a contact with conversation history", nil)
		}

		if err := app.Delete(contact); err != nil {
			return e.InternalServerError("failed to delete contact", err)
		}
		return e.JSON(http.StatusOK, map[string]any{"id": contact.Id})
	}
}

// HandleAccountCreate connects a WhatsApp number to the current org.
func HandleAccountCreate(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		if !access.CanManageData() {
			return e.ForbiddenError("only the owner or a superadmin can manage numbers", nil)
		}

		var body struct {
			DisplayName   string `json:"display_name" form:"display_name"`
			PhoneNumberID string `json:"phone_number_id" form:"phone_number_id"`
			AccessToken   string `json:"access_token" form:"access_token"`
			VerifyToken   string `json:"verify_token" form:"verify_token"`
			WabaID        string `json:"waba_id" form:"waba_id"`
			Status        string `json:"status" form:"status"`
			TeamID        string `json:"team_id" form:"team_id"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("Invalid request body", err)
		}
		body.PhoneNumberID = strings.TrimSpace(body.PhoneNumberID)
		if body.PhoneNumberID == "" {
			return e.BadRequestError("phone_number_id is required", nil)
		}
		if body.Status == "" {
			body.Status = accountStatusDisconnected
		}
		if body.Status != accountStatusConnected && body.Status != accountStatusDisconnected {
			return e.BadRequestError("invalid status", nil)
		}
		teamID := ""
		if body.TeamID != "" {
			var err error
			teamID, err = resolveTeam(app, access, store.RoleAgent, body.TeamID)
			if err != nil {
				return e.BadRequestError(err.Error(), nil)
			}
		}

		if _, err := store.FindWhatsAppAccountByPhoneNumberID(app, body.PhoneNumberID); err == nil {
			return e.BadRequestError("a number with this phone_number_id already exists", nil)
		}

		accountsCol, err := app.FindCollectionByNameOrId("whatsapp_accounts")
		if err != nil {
			return e.InternalServerError("accounts collection not found", err)
		}
		acc := core.NewRecord(accountsCol)
		acc.Set("org", access.OrgID)
		acc.Set("display_name", body.DisplayName)
		acc.Set("phone_number_id", body.PhoneNumberID)
		acc.Set("access_token", body.AccessToken)
		acc.Set("verify_token", body.VerifyToken)
		acc.Set("waba_id", strings.TrimSpace(body.WabaID))
		acc.Set("status", body.Status)
		acc.Set("team", teamID)
		if err := app.Save(acc); err != nil {
			return e.BadRequestError("failed to create number", err)
		}

		return e.JSON(http.StatusOK, waAccountFromRecord(app, acc))
	}
}

// HandleAccountUpdate updates a WhatsApp number in the current org.
func HandleAccountUpdate(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		if !access.CanManageData() {
			return e.ForbiddenError("only the owner or a superadmin can manage numbers", nil)
		}

		acc, err := store.FindOrgRecord(app, access.OrgID, "whatsapp_accounts", e.Request.PathValue("id"))
		if err != nil {
			return e.NotFoundError("number not found", nil)
		}

		var body struct {
			DisplayName   string `json:"display_name" form:"display_name"`
			PhoneNumberID string `json:"phone_number_id" form:"phone_number_id"`
			AccessToken   string `json:"access_token" form:"access_token"`
			VerifyToken   string `json:"verify_token" form:"verify_token"`
			WabaID        string `json:"waba_id" form:"waba_id"`
			Status        string `json:"status" form:"status"`
			TeamID        string `json:"team_id" form:"team_id"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("Invalid request body", err)
		}

		if body.PhoneNumberID != "" {
			newPhone := strings.TrimSpace(body.PhoneNumberID)
			// Reject if the new phone_number_id is already claimed by another account.
			if existing, err := store.FindWhatsAppAccountByPhoneNumberID(app, newPhone); err == nil && existing.Id != acc.Id {
				return e.BadRequestError("a number with this phone_number_id already exists", nil)
			}
			acc.Set("phone_number_id", newPhone)
		}
		if body.DisplayName != "" {
			acc.Set("display_name", body.DisplayName)
		}
		if body.AccessToken != "" {
			acc.Set("access_token", body.AccessToken)
		}
		if body.VerifyToken != "" {
			acc.Set("verify_token", body.VerifyToken)
		}
		if body.WabaID != "" {
			acc.Set("waba_id", strings.TrimSpace(body.WabaID))
		}
		if body.Status != "" {
			if body.Status != accountStatusConnected && body.Status != accountStatusDisconnected {
				return e.BadRequestError("invalid status", nil)
			}
			acc.Set("status", body.Status)
		}
		if body.TeamID != "" {
			teamID, err := resolveTeam(app, access, store.RoleAgent, body.TeamID)
			if err != nil {
				return e.BadRequestError(err.Error(), nil)
			}
			acc.Set("team", teamID)
		}

		if err := app.Save(acc); err != nil {
			return e.BadRequestError("failed to update number", err)
		}

		return e.JSON(http.StatusOK, waAccountFromRecord(app, acc))
	}
}

// HandleAccountDelete disconnects a WhatsApp number with no conversation history.
func HandleAccountDelete(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		if !access.CanManageData() {
			return e.ForbiddenError("only the owner or a superadmin can manage numbers", nil)
		}

		acc, err := store.FindOrgRecord(app, access.OrgID, "whatsapp_accounts", e.Request.PathValue("id"))
		if err != nil {
			return e.NotFoundError("number not found", nil)
		}

		count, err := countConversationsFor(app, "conversations", "whatsapp_account", access.OrgID, acc.Id)
		if err != nil {
			return e.InternalServerError("Failed to check conversations", err)
		}
		if count > 0 {
			return e.BadRequestError("cannot delete a number that has conversations", nil)
		}

		if err := app.Delete(acc); err != nil {
			return e.InternalServerError("failed to delete number", err)
		}
		return e.JSON(http.StatusOK, map[string]any{"id": acc.Id})
	}
}
