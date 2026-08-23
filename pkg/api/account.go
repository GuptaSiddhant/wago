package api

import (
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// accountUpdateRequest is the editable subset of the logged-in user's own
// profile. Email and password changes are intentionally excluded; they need
// verification/reset flows of their own.
type accountUpdateRequest struct {
	Name string `json:"name"`
}

// HandleProfileUpdate lets an authenticated user update their own profile
// (display name). It never touches other users' records.
func HandleProfileUpdate(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			return e.UnauthorizedError("authentication required", nil)
		}

		var body accountUpdateRequest
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("Invalid request body", err)
		}
		name := strings.TrimSpace(body.Name)
		if name == "" {
			return e.BadRequestError("name is required", nil)
		}

		e.Auth.Set("name", name)
		if err := app.Save(e.Auth); err != nil {
			return e.InternalServerError("Failed to update account", err)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"id":    e.Auth.Id,
			"name":  e.Auth.GetString("name"),
			"email": e.Auth.Email(),
		})
	}
}
