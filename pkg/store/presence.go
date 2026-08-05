package store

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// EnsurePresenceField adds the last_active_at presence marker to org_members.
func EnsurePresenceField(app core.App) error {
	col, err := app.FindCollectionByNameOrId("org_members")
	if err != nil {
		return err
	}
	ensureField(col, &core.DateField{Name: "last_active_at"})
	return app.Save(col)
}

// TouchPresence marks the member's last_active_at as now. Called by the
// frontend heartbeat while the user is using the app.
func TouchPresence(app core.App, orgID, userID string) error {
	member, err := FindOrgMembership(app, orgID, userID)
	if err != nil {
		return err
	}
	member.Set("last_active_at", time.Now())
	return app.Save(member)
}

// IsUserActive reports whether the member pinged presence within the window.
func IsUserActive(app core.App, orgID, userID string, within time.Duration) (bool, error) {
	member, err := FindOrgMembership(app, orgID, userID)
	if err != nil {
		return false, err
	}
	last := member.GetDateTime("last_active_at").Time()
	if last.IsZero() {
		return false, nil
	}
	return time.Since(last) <= within, nil
}

// EnsureUserPhoneField adds an optional phone field to the users collection,
// used for best-effort WhatsApp notifications to a user's own number.
func EnsureUserPhoneField(app core.App) error {
	col, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}
	ensureField(col, &core.TextField{Name: "phone"})
	return app.Save(col)
}
