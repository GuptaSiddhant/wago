package push

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/SherClockHolmes/webpush-go"

	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
)

// Payload is the JSON body delivered to the browser's push service and read by
// the service worker.
type Payload struct {
	Title          string `json:"title"`
	Body           string `json:"body"`
	ConversationID string `json:"conversation_id"`
	Org            string `json:"org"`
}

// Sender sends Web Push notifications to a user's registered devices. Sending
// is best-effort: expired subscriptions are removed, but errors never fail the
// caller.
type Sender struct {
	app     core.App
	subject string
}

// NewSender builds a push sender. subject is the VAPID "sub" contact (an email
// or URL the push service can reach).
func NewSender(app core.App, subject string) *Sender {
	return &Sender{app: app, subject: subject}
}

// Send delivers the payload to every device subscribed by the user.
func (s *Sender) Send(ctx context.Context, orgID, userID string, payload Payload) {
	subs, err := store.ListPushSubscriptions(s.app, orgID, userID)
	if err != nil || len(subs) == 0 {
		return
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	publicKey, privateKey, err := store.GetVAPIDKeys(s.app, webpush.GenerateVAPIDKeys)
	if err != nil {
		log.Printf("push: no VAPID keys: %v", err)
		return
	}

	for _, rec := range subs {
		endpoint := rec.GetString("endpoint")
		if endpoint == "" {
			continue
		}
		sub := &webpush.Subscription{
			Endpoint: endpoint,
			Keys: webpush.Keys{
				Auth:   rec.GetString("auth"),
				P256dh: rec.GetString("p256dh"),
			},
		}
		resp, err := webpush.SendNotificationWithContext(ctx, data, sub, &webpush.Options{
			Subscriber:      s.subject,
			VAPIDPublicKey:  publicKey,
			VAPIDPrivateKey: privateKey,
			TTL:             7 * 24 * 60, // 7 days so phones wake up even while off
			Urgency:         webpush.UrgencyHigh,
		})
		if err != nil {
			log.Printf("push: failed to notify %s: %v", endpoint, err)
			continue
		}
		// A 404/410 means the subscription no longer exists (e.g. the user
		// removed it or it expired) — drop it so we stop retrying.
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
			if err := store.DeletePushSubscriptionByID(s.app, orgID, rec.Id); err != nil {
				log.Printf("push: failed to prune expired subscription: %v", err)
			}
		} else if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			log.Printf("push: push service returned %d for %s", resp.StatusCode, endpoint)
		}
	}
}
