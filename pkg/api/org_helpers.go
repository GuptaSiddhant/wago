package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// rawSliceString unmarshals a JSON-field value into the given string slice.
// It is a no-op when the field is unset or not a JSON array.
func rawSliceString(rec *core.Record, field string, out *[]string) error {
	raw := rec.Get(field)
	if raw == nil {
		return nil
	}
	if b, ok := raw.(json.RawMessage); ok {
		return json.Unmarshal(b, out)
	}
	switch v := raw.(type) {
	case []string:
		*out = v
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				*out = append(*out, s)
			}
		}
	}
	return nil
}

// orgProfilePictureURL builds the org-scoped URL that streams the org's stored
// profile picture, or "" when none is set. The orgs collection is locked down
// to the raw PocketBase API, so pictures are served through the Wago API with
// org access checks.
func orgProfilePictureURL(org *core.Record) string {
	if org.GetString("profile_picture") == "" {
		return ""
	}
	return "/api/wa/orgs/" + org.Id + "/picture"
}

// HandleOrgPicture streams an org's stored profile picture. Access is scoped
// to members of the org (or superusers). Because <img> tags cannot send the
// Authorization header, the session token is also accepted via the
// `authorization` query parameter (validated with the same rules as the header).
func HandleOrgPicture(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			if q := strings.TrimSpace(e.Request.URL.Query().Get("authorization")); q != "" {
				record, err := app.FindAuthRecordByToken(q, core.TokenTypeAuth)
				if err == nil && record != nil {
					e.Auth = record
				}
			}
		}

		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		org, err := app.FindRecordById("orgs", access.OrgID)
		if err != nil {
			return e.NotFoundError("organization not found", nil)
		}

		filename := org.GetString("profile_picture")
		if filename == "" {
			return e.NotFoundError("organization has no profile picture", nil)
		}

		fsys, err := app.NewFilesystem()
		if err != nil {
			return e.InternalServerError("Filesystem initialization failure", err)
		}
		defer fsys.Close()

		return fsys.Serve(e.Response, e.Request, org.BaseFilesPath()+"/"+filename, filename)
	}
}

// HandleOrgGet returns the detailed business profile of an org for its members.
func HandleOrgGet(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		org, err := app.FindRecordById("orgs", access.OrgID)
		if err != nil {
			return e.NotFoundError("organization not found", nil)
		}

		return e.JSON(http.StatusOK, orgToSummary(org, access.Role))
	}
}