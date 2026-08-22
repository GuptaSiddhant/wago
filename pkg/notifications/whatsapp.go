package notifications

import (
	"context"
	"log"
	"time"

	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
)

// sendWhatsApp delivers a best-effort WhatsApp notification to an inactive
// assignee. It only fires when an approved notification template is configured
// AND the user has a phone number on file. All failures are logged, never
// propagated, so a broken notification path can't affect message handling.
func (n *Notifier) sendWhatsApp(ctx context.Context, app core.App, orgID, convID string, user *core.Record, contact, preview string) {
	cfg := n.config(app)
	if cfg.WA_NotificationTemplate == "" {
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

	// Add timeout for WhatsApp API call
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if _, err := n.client.SendTemplate(
		ctx,
		acc.GetString("access_token"),
		acc.GetString("phone_number_id"),
		phone,
		cfg.WA_NotificationTemplate,
		"en_US",
		params,
		nil,
	); err != nil {
		log.Printf("notifications: whatsapp: template send failed for %s: %v", contact, err)
	}
}
