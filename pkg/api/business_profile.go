package api

import (
	"net/http"

	"github.com/guptasiddhant/wago/pkg/meta"
	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
)

// HandleAccountBusinessProfile fetches the WhatsApp business profile currently
// set on a phone number, so the org's own profile page can show the source of
// truth and warn when they differ.
func HandleAccountBusinessProfile(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		acc, err := store.FindOrgRecord(app, access.OrgID, "whatsapp_accounts", e.Request.PathValue("id"))
		if err != nil {
			return e.NotFoundError("number not found", nil)
		}

		token := acc.GetString("access_token")
		phoneID := acc.GetString("phone_number_id")
		if token == "" || phoneID == "" {
			return e.JSON(http.StatusOK, map[string]any{
				"ok":    false,
				"error": "number is missing an access token or phone_number_id",
			})
		}

		profile, err := metaClient.GetBusinessProfile(e.Request.Context(), token, phoneID)
		if err != nil {
			return e.JSON(http.StatusOK, map[string]any{
				"ok":    false,
				"error": err.Error(),
			})
		}

		return e.JSON(http.StatusOK, map[string]any{"ok": true, "profile": profile})
	}
}

// HandleAccountBusinessProfileSync pushes the org's business profile fields
// (about, address, description, email, websites, vertical) to the phone
// number's WhatsApp business profile via Meta's Graph API. Only filled-in org
// fields are sent, so the remaining Meta fields are left untouched.
func HandleAccountBusinessProfileSync(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		if !access.CanManageData() {
			return e.ForbiddenError("only the owner or a superadmin can sync the business profile", nil)
		}

		acc, err := store.FindOrgRecord(app, access.OrgID, "whatsapp_accounts", e.Request.PathValue("id"))
		if err != nil {
			return e.NotFoundError("number not found", nil)
		}

		token := acc.GetString("access_token")
		phoneID := acc.GetString("phone_number_id")
		if token == "" || phoneID == "" {
			return e.BadRequestError("number is missing an access token or phone_number_id", nil)
		}

		org, err := app.FindRecordById("orgs", access.OrgID)
		if err != nil {
			return e.InternalServerError("org not found", err)
		}

		var websites []string
		_ = rawSliceString(org, "websites", &websites)

		profile := &meta.BusinessProfile{
			About:       org.GetString("about"),
			Address:     org.GetString("address"),
			Description: org.GetString("description"),
			Email:       org.GetString("email"),
			Websites:    websites,
			Vertical:    org.GetString("vertical"),
		}

		if err := metaClient.UpdateBusinessProfile(e.Request.Context(), token, phoneID, profile); err != nil {
			return e.BadRequestError("Failed to sync the business profile with Meta", err)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"ok":      true,
			"message": "Business profile synced to WhatsApp.",
		})
	}
}