package notifications

import (
	"context"
	"log"

	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
)

// sendWhatsApp delivers a best-effort WhatsApp notification to an inactive
// assignee. It only fires when an approved notification template is configured
// AND the user has a phone number on file. All failures are logged, never
// propagated, so a broken notification path can't affect message handling.
func (n *Notifier) sendWhatsApp(app core.App, orgID, convID string, user *core.Record, contact, preview string) {
	if n.cfg.WA_NotificationTemplate == "" {
		return // WhatsApp notifications disabled
	}
	phone := user.GetString("phone")
	if phone == "" {
		return // no phone on file for this user
	}

	conv, err := app.FindRecordById("conversations", convID)
	if err != nil {
		log.Printf("notifications: whatsapp: conversation not found: %v", err)
		return
	}
	acc, err := store.FindWhatsAppAccountByID(app, orgID, conv.GetString("whatsapp_account"))
	if err != nil {
		log.Printf("notifications: whatsapp: account not found: %v", err)
		return
	}

	params := []map[string]any{
		{"type": "text", "text": preview},
	}

	if _, err := n.client.SendTemplate(
		context.Background(),
		acc.GetString("access_token"),
		acc.GetString("phone_number_id"),
		phone,
		n.cfg.WA_NotificationTemplate,
		"en_US",
		params,
	); err != nil {
		log.Printf("notifications: whatsapp: template send failed for %s: %v", contact, err)
	}
}
