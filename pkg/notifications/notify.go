package notifications

import (
	"log"
	"time"

	"github.com/guptasiddhant/wago/pkg/meta"
	"github.com/guptasiddhant/wago/pkg/store"
	"github.com/guptasiddhant/wago/pkg/utils"

	"github.com/pocketbase/pocketbase/core"
)

// activeWindow is how recently a member must have pinged presence to be
// considered "active" (and skip email/WhatsApp delivery in favour of the
// desktop push that the running frontend shows).
const activeWindow = 5 * time.Minute

// Notifier triggers notifications and delivers them to inactive users.
type Notifier struct {
	cfg    *utils.AppConfig
	client *meta.Client
}

// NewNotifier builds a Notifier from configuration. It is safe to share.
func NewNotifier(cfg *utils.AppConfig) *Notifier {
	return &Notifier{cfg: cfg, client: meta.NewClient()}
}

// Trigger is called from the webhook after an inbound message is persisted and
// its conversation unread counter incremented. If the conversation is assigned
// to a user, it records a notification for them and, if they are inactive,
// delivers it asynchronously via email and/or WhatsApp.
func (n *Notifier) Trigger(app core.App, orgID, convID, assigneeID, preview string) {
	if assigneeID == "" {
		return // not assigned to anyone yet — nothing to notify
	}

	ok, err := store.CanCreateNotification(app, orgID, assigneeID, convID)
	if err != nil || !ok {
		return // throttled: a recent unread notification already exists
	}
	if _, err := store.CreateNotification(app, orgID, assigneeID, convID, store.NotificationInbound, preview); err != nil {
		log.Printf("notifications: failed to create notification: %v", err)
		return
	}

	active, err := store.IsUserActive(app, orgID, assigneeID, activeWindow)
	if err != nil {
		log.Printf("notifications: presence check failed: %v", err)
		return
	}
	if active {
		return // online — the frontend desktop push will surface it
	}

	// Deliver outside the request so the webhook can return to Meta promptly.
	go n.deliver(app, orgID, assigneeID, convID, preview)
}

// deliver sends email and WhatsApp notifications to an inactive assignee.
func (n *Notifier) deliver(app core.App, orgID, userID, convID, preview string) {
	user, err := app.FindRecordById("users", userID)
	if err != nil {
		log.Printf("notifications: user not found: %v", err)
		return
	}

	contact := n.contactName(app, convID)

	if err := n.sendEmail(app, user.Email(), contact, preview); err != nil {
		log.Printf("notifications: email delivery failed: %v", err)
	}
	n.sendWhatsApp(app, orgID, convID, user, contact, preview)
}

// contactName resolves the human-readable contact associated with a conversation.
func (n *Notifier) contactName(app core.App, convID string) string {
	conv, err := app.FindRecordById("conversations", convID)
	if err != nil {
		return "a customer"
	}
	contact, err := store.FindOrgRecord(app, conv.GetString("org"), "contacts", conv.GetString("contact"))
	if err != nil {
		return "a customer"
	}
	if name := contact.GetString("name"); name != "" {
		return name
	}
	if phone := contact.GetString("phone"); phone != "" {
		return phone
	}
	return "a customer"
}