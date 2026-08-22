package store

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
)

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
	return ensureFields(app, "users", &core.TextField{Name: "phone"})
}
