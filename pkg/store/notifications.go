package store

import (
	"database/sql"
	"log"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Notification kinds.
const (
	NotificationInbound = "inbound"
)

// notifyCooldown is how long an unread notification suppresses further
// notifications for the same user + conversation, so a burst of messages in a
// single chat doesn't spam the user (or emails) repeatedly.
const notifyCooldown = 10 * time.Minute

// EnsureNotificationsCollection creates the org-scoped notifications collection
// that stores unread push/email/WhatsApp alerts for users.
func EnsureNotificationsCollection(app core.App) error {
	if _, err := app.FindCollectionByNameOrId("notifications"); err != nil {
		usersCol, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		orgsCol, err := app.FindCollectionByNameOrId("orgs")
		if err != nil {
			return err
		}
		convsCol, err := app.FindCollectionByNameOrId("conversations")
		if err != nil {
			return err
		}

		col := core.NewBaseCollection("notifications")
		// Notifications are org/user scoped and only reachable through the
		// scoped API (and PB superusers which bypass rules).
		col.ListRule = nil
		col.ViewRule = nil
		col.CreateRule = nil
		col.UpdateRule = nil
		col.DeleteRule = nil

		col.Fields.Add(
			&core.RelationField{
				Name:          "org",
				CollectionId:  orgsCol.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.RelationField{
				Name:          "user",
				CollectionId:  usersCol.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.RelationField{
				Name:          "conversation",
				CollectionId:  convsCol.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.TextField{Name: "kind"},
			&core.TextField{Name: "body"},
			&core.BoolField{Name: "read"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		col.AddIndex("idx_notifications_user_created", false, "user, created", "")

		if err := app.Save(col); err != nil {
			return err
		}
		log.Println("Auto-created 'notifications' collection")
	}
	return nil
}

// CanCreateNotification reports whether a new notification for the given
// user + conversation should be created (i.e. no recent unread one exists).
func CanCreateNotification(app core.App, orgID, userID, convID string) (bool, error) {
	rec, err := app.FindFirstRecordByFilter("notifications",
		"org = {:org} && user = {:user} && conversation = {:conv} && read = false",
		dbx.Params{"org": orgID, "user": userID, "conv": convID})
	if err != nil {
		if err == sql.ErrNoRows {
			return true, nil
		}
		return false, err
	}
	created := rec.GetDateTime("created").Time()
	if time.Since(created) < notifyCooldown {
		return false, nil
	}
	return true, nil
}

// CreateNotification inserts a notification record for a user.
func CreateNotification(app core.App, orgID, userID, convID, kind, body string) (*core.Record, error) {
	col, err := app.FindCollectionByNameOrId("notifications")
	if err != nil {
		return nil, err
	}
	rec := core.NewRecord(col)
	rec.Set("org", orgID)
	rec.Set("user", userID)
	rec.Set("conversation", convID)
	rec.Set("kind", kind)
	rec.Set("body", body)
	rec.Set("read", false)
	if err := app.Save(rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// ListUserNotifications returns the newest notifications for a user in an org.
func ListUserNotifications(app core.App, orgID, userID string, limit int) ([]*core.Record, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return app.FindRecordsByFilter("notifications",
		"org = {:org} && user = {:user}", "-created", limit, 0,
		DbxParams(map[string]any{"org": orgID, "user": userID}))
}

// UnreadNotificationCount returns the count of unread notifications for a user.
func UnreadNotificationCount(app core.App, orgID, userID string) (int, error) {
	recs, err := app.FindRecordsByFilter("notifications",
		"org = {:org} && user = {:user} && read = false", "", 0, 0,
		DbxParams(map[string]any{"org": orgID, "user": userID}))
	if err != nil {
		return 0, err
	}
	return len(recs), nil
}

// MarkAllNotificationsRead marks every notification for a user in an org as
// read. Returns the number of notifications that were updated.
func MarkAllNotificationsRead(app core.App, orgID, userID string) (int, error) {
	recs, err := app.FindRecordsByFilter("notifications",
		"org = {:org} && user = {:user} && read = false", "", 0, 0,
		DbxParams(map[string]any{"org": orgID, "user": userID}))
	if err != nil {
		return 0, err
	}
	for _, rec := range recs {
		rec.Set("read", true)
		if err := app.Save(rec); err != nil {
			return 0, err
		}
	}
	return len(recs), nil
}
