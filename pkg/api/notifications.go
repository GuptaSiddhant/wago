package api

import (
	"net/http"
	"strconv"

	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

type notificationDTO struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	Body           string `json:"body"`
	Read           bool   `json:"read"`
	ConversationID string `json:"conversation_id"`
	ContactName    string `json:"contact_name"`
	Created        string `json:"created"`
}

func notificationToDTO(app core.App, orgID string, rec *core.Record) notificationDTO {
	dto := notificationDTO{
		ID:             rec.Id,
		Kind:           rec.GetString("kind"),
		Body:           rec.GetString("body"),
		Read:           rec.GetBool("read"),
		ConversationID: rec.GetString("conversation"),
		Created:        rec.GetDateTime("created").Time().UTC().Format(types.DefaultDateLayout),
	}

	convID := rec.GetString("conversation")
	if conv, err := store.FindOrgRecord(app, orgID, "conversations", convID); err == nil {
		if contact, err := store.FindOrgRecord(app, orgID, "contacts", conv.GetString("contact")); err == nil {
			if name := contact.GetString("name"); name != "" {
				dto.ContactName = name
			} else {
				dto.ContactName = contact.GetString("phone")
			}
		}
	}
	return dto
}

// HandleNotificationsList lists the current user's notifications.
func HandleNotificationsList(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		limit, _ := strconv.Atoi(e.Request.URL.Query().Get("limit"))
		records, err := store.ListUserNotifications(app, access.OrgID, e.Auth.Id, limit)
		if err != nil {
			return e.InternalServerError("Failed to list notifications", err)
		}

		items := make([]notificationDTO, 0, len(records))
		for _, r := range records {
			items = append(items, notificationToDTO(app, access.OrgID, r))
		}
		return e.JSON(http.StatusOK, map[string]any{"items": items})
	}
}

// HandleNotificationsUnreadCount returns the current user's unread count.
func HandleNotificationsUnreadCount(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		count, err := store.UnreadNotificationCount(app, access.OrgID, e.Auth.Id)
		if err != nil {
			return e.InternalServerError("Failed to count notifications", err)
		}
		return e.JSON(http.StatusOK, map[string]any{"count": count})
	}
}

// HandleNotificationsRead marks all of the current user's notifications read.
func HandleNotificationsRead(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		updated, err := store.MarkAllNotificationsRead(app, access.OrgID, e.Auth.Id)
		if err != nil {
			return e.InternalServerError("Failed to mark notifications read", err)
		}
		return e.JSON(http.StatusOK, map[string]any{"updated": updated})
	}
}

// HandlePresence records the current user's presence heartbeat for the org.
func HandlePresence(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		if err := store.TouchPresence(app, access.OrgID, e.Auth.Id); err != nil {
			return e.InternalServerError("Failed to update presence", err)
		}
		return e.NoContent(http.StatusNoContent)
	}
}