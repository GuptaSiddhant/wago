package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
)

// waAccountDTO is a WhatsApp account listing for the settings page.
type waAccountDTO struct {
	ID            string `json:"id"`
	DisplayName   string `json:"display_name"`
	PhoneNumberID string `json:"phone_number_id"`
	Status        string `json:"status"`
}

// teamMemberDTO is an org member listing for the settings page.
type teamMemberDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
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
			items = append(items, contactDTO{
				ID:    r.Id,
				Name:  r.GetString("name"),
				Phone: r.GetString("phone"),
			})
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
			items = append(items, teamMemberDTO{
				ID:    user.Id,
				Name:  user.GetString("name"),
				Email: user.Email(),
				Role:  m.GetString("role"),
			})
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
			items = append(items, waAccountDTO{
				ID:            r.Id,
				DisplayName:   r.GetString("display_name"),
				PhoneNumberID: r.GetString("phone_number_id"),
				Status:        r.GetString("status"),
			})
		}

		return e.JSON(http.StatusOK, map[string]any{"items": items})
	}
}
