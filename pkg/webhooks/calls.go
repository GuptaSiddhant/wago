package webhooks

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/guptasiddhant/wago/pkg/callhub"
	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
)

// inboundCallRequest is the payload POSTed by the call provider. This is the
// first integration of a talking provider (e.g. Meta's calls webhook shape). It
// carries the phone number id and the caller's number; a display name is
// optional. As the provider mute becomes real, only phone/status mapping here
// will need adjusting, the store and hub handling stays the same.
type inboundCallRequest struct {
	PhoneNumberID  string `json:"phone_number_id"`
	FromID         string `json:"from"`
	DisplayName    string `json:"display_name,omitempty"`
	Status         string `json:"status,omitempty"`
	ProviderCallID string `json:"provider_call_id,omitempty"`
}

// HandleInboundCall routes an inbound "ringing" event to the org that owns the
// phone number, materializes a conversation if needed, records a call, and
// publishes an "incoming" event so a connected agent sees the answer banner.
func HandleInboundCall() func(re *core.RequestEvent) error {
	return func(re *core.RequestEvent) error {
		var req inboundCallRequest
		if err := re.BindBody(&req); err != nil {
			return re.BadRequestError("Invalid payload", err)
		}
		if req.PhoneNumberID == "" || req.FromID == "" {
			return re.BadRequestError("phone_number_id and from are required", nil)
		}

		app := re.App
		account, err := store.FindWhatsAppAccountByPhoneNumberID(app, req.PhoneNumberID)
		if err != nil {
			logf(app, slog.LevelWarn, "ignoring call for unknown phone_number_id %q", req.PhoneNumberID)
			return re.String(http.StatusOK, "EVENT_RECEIVED")
		}
		orgID := account.GetString("org")

		contact, err := store.UpsertContact(app, orgID, req.FromID, req.DisplayName)
		if err != nil {
			return re.InternalServerError("Failed to upsert contact", err)
		}

		ts := time.Now()
		conv, created, err := store.UpsertConversation(app, orgID, contact.Id, account.Id, ts)
		if err != nil {
			return re.InternalServerError("Failed to upsert conversation", err)
		}
		if created {
			if assignee, err := store.AssignConversationRR(app, conv); err != nil {
				logf(app, slog.LevelWarn, "failed to round-robin assign conversation %s: %v", conv.Id, err)
			} else if assignee != "" {
				logf(app, slog.LevelInfo, "round-robin assigned conversation %s to %s", conv.Id, assignee)
			}
		}

		call, err := store.CreateIncomingCall(app, conv, store.CallDirectionInbound,
			req.FromID, req.DisplayName, ts)
		if err != nil {
			return re.InternalServerError("Failed to create call", err)
		}

		callhub.DefaultHub.Publish(orgID, callhub.CallEvent{
			ID:        call.Id,
			Direction: call.GetString("direction"),
			State:     store.CallRinging,
			Phone:     call.GetString("phone"),
			Name:      call.GetString("peer_name"),
		})

		logf(app, slog.LevelInfo, "inbound call %s for org %s from %s", call.Id, orgID, req.FromID)
		return re.String(http.StatusOK, "EVENT_RECEIVED")
	}
}
