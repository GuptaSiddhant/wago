package api

import (
	"strings"

	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

// orgAccessFromRequest resolves the requested org for an authenticated request.
// The org is read from the "X-Org-Id" header, falling back to the "org" query param.
func orgAccessFromRequest(e *core.RequestEvent, app core.App) (*store.OrgAccess, *router.ApiError) {
	orgID := strings.TrimSpace(e.Request.Header.Get("X-Org-Id"))
	if orgID == "" {
		orgID = strings.TrimSpace(e.Request.URL.Query().Get("org"))
	}
	if orgID == "" {
		return nil, e.BadRequestError("missing organization: set the X-Org-Id header", nil)
	}

	access, err := store.ResolveOrgAccess(app, e.Auth, orgID)
	if err != nil {
		return nil, e.ForbiddenError(err.Error(), nil)
	}
	return access, nil
}
