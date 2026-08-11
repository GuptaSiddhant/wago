package api

import (
	"net/http"

	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
)

type loginRequest struct {
	Email    string `json:"email" form:"email"`
	Password string `json:"password" form:"password"`
}

type orgSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

// sessionResponse is returned by login and /auth/me.
type sessionResponse struct {
	Token     string       `json:"token,omitempty"`
	User      *userSummary `json:"user"`
	IsAdmin   bool         `json:"is_admin"`
	AIEnabled bool         `json:"ai_enabled"`
	Orgs      []orgSummary `json:"orgs"`
}

type userSummary struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// HandleLogin authenticates a user or superuser and returns a session.
func HandleLogin(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var body loginRequest
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("Invalid request body", err)
		}
		if body.Email == "" || body.Password == "" {
			return e.BadRequestError("email and password are required", nil)
		}

		result, err := store.Authenticate(app, body.Email, body.Password)
		if err != nil {
			return e.UnauthorizedError("Invalid credentials", nil)
		}

		session, err := buildSession(app, result)
		if err != nil {
			return e.InternalServerError("Failed to build session", err)
		}

		return e.JSON(http.StatusOK, session)
	}
}

// HandleMe returns the current session for an authenticated request.
func HandleMe(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		result := &store.AuthResult{
			Record:  e.Auth,
			Token:   "",
			IsAdmin: e.HasSuperuserAuth(),
		}
		session, err := buildSession(app, result)
		if err != nil {
			return e.InternalServerError("Failed to build session", err)
		}
		return e.JSON(http.StatusOK, session)
	}
}

func buildSession(app core.App, result *store.AuthResult) (*sessionResponse, error) {
	rec := result.Record

	orgs := []orgSummary{}

	if result.IsAdmin {
		all, err := store.AllOrgs(app)
		if err != nil {
			return nil, err
		}
		for _, o := range all {
			orgs = append(orgs, orgSummary{ID: o.Id, Name: o.GetString("name"), Role: store.RoleOwner})
		}
	} else {
		memberships, err := store.UserOrgs(app, rec.Id)
		if err != nil {
			return nil, err
		}
		for _, m := range memberships {
			orgID := m.GetString("org")
			orgRec, err := app.FindRecordById("orgs", orgID)
			if err != nil {
				continue
			}
			orgs = append(orgs, orgSummary{
				ID:   orgID,
				Name: orgRec.GetString("name"),
				Role: m.GetString("role"),
			})
		}
	}

	return &sessionResponse{
		Token:     result.Token,
		IsAdmin:   result.IsAdmin,
		AIEnabled: aiCfg.Enabled,
		User: &userSummary{
			ID:    rec.Id,
			Email: rec.Email(),
			Name:  rec.GetString("name"),
		},
		Orgs: orgs,
	}, nil
}
