package api

import (
	"io"
	"net/http"
	"strings"

	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// HandleOrgCreate lets a superadmin create an organization. Regular users
// cannot create orgs; they are invited into existing ones.
//
// The request is a multipart form. Alongside `name`, the WhatsApp Business
// Profile fields (`about`, `address`, `description`, `email`, `websites`,
// `vertical`) are accepted, plus an optional `profile_picture` file which is
// stored on the org's `profile_picture` file field.
func HandleOrgCreate(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if !e.HasSuperuserAuth() {
			return e.ForbiddenError("only a superadmin can create organizations", nil)
		}

		if err := e.Request.ParseMultipartForm(100 << 20); err != nil {
			return e.BadRequestError("Invalid multipart body", err)
		}

		name := strings.TrimSpace(e.Request.FormValue("name"))
		if name == "" {
			return e.BadRequestError("organization name is required", nil)
		}

		col, err := app.FindCollectionByNameOrId("orgs")
		if err != nil {
			return e.InternalServerError("orgs collection not found", err)
		}
		org := core.NewRecord(col)
		org.Set("name", name)

		applyOrgBusinessFields(org, e, false)

		if file, header, err := e.Request.FormFile("profile_picture"); err == nil {
			data, readErr := io.ReadAll(file)
			file.Close()
			if readErr != nil {
				return e.BadRequestError("Failed to read profile picture", readErr)
			}
			if len(data) == 0 {
				return e.BadRequestError("profile picture file is empty", nil)
			}
			pic, picErr := filesystem.NewFileFromBytes(data, header.Filename)
			if picErr != nil {
				return e.BadRequestError("Invalid profile picture", picErr)
			}
			org.Set("profile_picture", pic)
		}

		if err := app.Save(org); err != nil {
			return e.BadRequestError("failed to create organization", err)
		}

		return e.JSON(http.StatusCreated, orgToSummary(org, store.RoleOwner))
	}
}

// HandleOrgUpdate lets an org owner/admin (or a superuser) edit the current
// organization's details. The request is a multipart form accepting the same
// WhatsApp Business Profile fields as create, plus an optional `profile_picture`
// file replacement and a `remove_picture` flag to delete the stored one.
func HandleOrgUpdate(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		if !access.CanManage() {
			return e.ForbiddenError("you cannot edit this organization", nil)
		}

		if err := e.Request.ParseMultipartForm(100 << 20); err != nil {
			return e.BadRequestError("Invalid multipart body", err)
		}

		org, err := app.FindRecordById("orgs", access.OrgID)
		if err != nil {
			return e.NotFoundError("organization not found", nil)
		}

		if name := strings.TrimSpace(e.Request.FormValue("name")); name != "" {
			org.Set("name", name)
		}

		applyOrgBusinessFields(org, e, true)

		if e.Request.FormValue("remove_picture") == "1" {
			org.Set("profile_picture", "")
		}
		if file, header, err := e.Request.FormFile("profile_picture"); err == nil {
			data, readErr := io.ReadAll(file)
			file.Close()
			if readErr != nil {
				return e.BadRequestError("Failed to read profile picture", readErr)
			}
			if len(data) == 0 {
				return e.BadRequestError("profile picture file is empty", nil)
			}
			pic, picErr := filesystem.NewFileFromBytes(data, header.Filename)
			if picErr != nil {
				return e.BadRequestError("Invalid profile picture", picErr)
			}
			org.Set("profile_picture", pic)
		}

		if err := app.Save(org); err != nil {
			return e.BadRequestError("failed to update organization", err)
		}

		return e.JSON(http.StatusOK, orgToSummary(org, access.Role))
	}
}

// applyOrgBusinessFields copies the WhatsApp Business Profile fields from the
// multipart form onto the org record, keeping the Meta API's field names. When
// overwrite is true, empty values clear the stored fields (used on update);
// otherwise empty values are left untouched (used on create).
func applyOrgBusinessFields(org *core.Record, e *core.RequestEvent, overwrite bool) {
	about := strings.TrimSpace(e.Request.FormValue("about"))
	if overwrite || about != "" {
		org.Set("about", about)
	}
	address := strings.TrimSpace(e.Request.FormValue("address"))
	if overwrite || address != "" {
		org.Set("address", address)
	}
	description := strings.TrimSpace(e.Request.FormValue("description"))
	if overwrite || description != "" {
		org.Set("description", description)
	}
	email := strings.TrimSpace(e.Request.FormValue("email"))
	if overwrite || email != "" {
		org.Set("email", email)
	}
	if websites := e.Request.Form["websites"]; len(websites) > 0 {
		cleaned := make([]string, 0, len(websites))
		for _, w := range websites {
			if w = strings.TrimSpace(w); w != "" {
				cleaned = append(cleaned, w)
			}
		}
		if overwrite || len(cleaned) > 0 {
			org.Set("websites", cleaned)
		}
	}
	if vertical := strings.TrimSpace(e.Request.FormValue("vertical")); overwrite || vertical != "" {
		org.Set("vertical", vertical)
	}
}

// orgToSummary builds the org summary DTO from an org record.
func orgToSummary(org *core.Record, role string) orgSummary {
	var websites []string
	if raw := org.Get("websites"); raw != nil {
		_ = rawSliceString(org, "websites", &websites)
	}
	return orgSummary{
		ID:          org.Id,
		Name:        org.GetString("name"),
		Role:        role,
		About:       org.GetString("about"),
		Address:     org.GetString("address"),
		Description: org.GetString("description"),
		Email:       org.GetString("email"),
		Websites:    websites,
		Vertical:    org.GetString("vertical"),
		PictureURL:  orgProfilePictureURL(org),
	}
}
