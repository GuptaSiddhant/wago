package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/guptasiddhant/wago/pkg/callhub"
	"github.com/guptasiddhant/wago/pkg/rtc"
	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
)

var rtcManager = rtc.NewManager()

// callDTO mirrors a voice_calls record for the API.
type callDTO struct {
	ID           string `json:"id"`
	Conversation string `json:"conversation_id"`
	ContactID    string `json:"contact_id"`
	AccountID    string `json:"account_id"`
	Direction    string `json:"direction"`
	Status       string `json:"status"`
	Phone        string `json:"phone"`
	Name         string `json:"name,omitempty"`
	Duration     int    `json:"duration"`
	StartedAt    string `json:"started_at,omitempty"`
	EndedAt      string `json:"ended_at,omitempty"`
	Created      string `json:"created"`
}

func callFromRecord(r *core.Record) callDTO {
	return callDTO{
		ID:           r.Id,
		Conversation: r.GetString("conversation"),
		ContactID:    r.GetString("contact"),
		AccountID:    r.GetString("account"),
		Direction:    r.GetString("direction"),
		Status:       r.GetString("status"),
		Phone:        r.GetString("phone"),
		Name:         r.GetString("peer_name"),
		Duration:     r.GetInt("duration"),
		StartedAt:    fmtDateTime(r.GetDateTime("started_at")),
		EndedAt:      fmtDateTime(r.GetDateTime("ended_at")),
		Created:      fmtDateTime(r.GetDateTime("created")),
	}
}

// HandleCallCreate starts an outbound call to a contact inside a conversation.
// It creates the call record (state = ringing) and returns it; the browser
// connects media through /calls/{id}/signal.
func HandleCallCreate(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		convID := strings.TrimSpace(e.Request.FormValue("conversation_id"))
		if convID == "" {
			return e.BadRequestError("conversation_id is required", nil)
		}
		conv, err := store.FindOrgRecord(app, access.OrgID, "conversations", convID)
		if err != nil {
			return e.NotFoundError("conversation not found", nil)
		}

		contact, err := store.FindOrgRecord(app, access.OrgID, "contacts", conv.GetString("contact"))
		if err != nil {
			return e.InternalServerError("contact not found", err)
		}

		call, err := store.CreateIncomingCall(app, conv, store.CallDirectionOutbound,
			contact.GetString("phone"), contact.GetString("name"), time.Now())
		if err != nil {
			return e.InternalServerError("Failed to start call", err)
		}

		callhub.DefaultHub.Publish(access.OrgID, callhub.CallEvent{
			ID:        call.Id,
			Direction: call.GetString("direction"),
			State:     call.GetString("status"),
			Phone:     call.GetString("phone"),
			Name:      call.GetString("peer_name"),
		})

		return e.JSON(http.StatusOK, callFromRecord(call))
	}
}

// callSignalRequest is the browser's WebRTC offer for a call.
type callSignalRequest struct {
	Offer string `json:"sdp" form:"sdp"`
}

// HandleCallSignal turns the browser offer into a connected media session. It
// answers the WebRTC offer through the pion bridge and returns the SDP answer.
func HandleCallSignal(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		callID := e.Request.PathValue("id")

		var body callSignalRequest
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("Invalid request body", err)
		}
		if body.Offer == "" {
			return e.BadRequestError("an SDP offer is required", nil)
		}

		call, err := store.FindOrgCall(app, access.OrgID, callID)
		if err != nil {
			return e.NotFoundError("call not found", nil)
		}

		// Mark the call as active the first time media is negotiated.
		if call.GetString("status") != store.CallActive {
			if err := store.SetCallStatus(app, callID, store.CallActive); err != nil {
				log.Printf("calls: failed to mark call %s active: %v", callID, err)
			} else {
				call.Set("status", store.CallActive)
				callhub.DefaultHub.Publish(access.OrgID, callhub.CallEvent{
					ID:        callID,
					Direction: call.GetString("direction"),
					State:     store.CallActive,
					Phone:     call.GetString("phone"),
					Name:      call.GetString("peer_name"),
				})
			}
		}

		bridge, ok := rtcManager.Get(callID)
		if !ok {
			bridge, err = rtcManager.NewCall(callID)
			if err != nil {
				return e.InternalServerError("Failed to create media session", err)
			}
		}

		answer, err := bridge.Answer(body.Offer)
		if err != nil {
			_ = rtcManager.End(callID)
			return e.BadRequestError("Failed to negotiate the call via WebRTC", err)
		}

		return e.JSON(http.StatusOK, map[string]any{"sdp": answer})
	}
}

// HandleCallEnd hangs up a call, tearing down its media session.
func HandleCallEnd(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		callID := e.Request.PathValue("id")
		call, err := store.FindOrgCall(app, access.OrgID, callID)
		if err != nil {
			return e.NotFoundError("call not found", nil)
		}
		_ = rtcManager.End(callID)
		status := store.CallEnded
		if call.GetString("status") == store.CallRinging {
			status = store.CallMissed
		}
		if err := store.SetCallStatus(app, callID, status); err != nil {
			return e.InternalServerError("Failed to end call", err)
		}
		callhub.DefaultHub.Publish(access.OrgID, callhub.CallEvent{
			ID:        callID,
			Direction: call.GetString("direction"),
			State:     status,
			Phone:     call.GetString("phone"),
			Name:      call.GetString("peer_name"),
		})
		return e.JSON(http.StatusOK, map[string]any{"id": callID, "status": status})
	}
}

// HandleConversationCalls lists recent calls against a conversation.
func HandleConversationCalls(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		convID := e.Request.PathValue("id")
		limit := parseLimit(e.Request.URL.Query().Get("limit"))
		records, err := store.ListConversationCalls(app, access.OrgID, convID, limit)
		if err != nil {
			return e.InternalServerError("Failed to list calls", err)
		}
		items := make([]callDTO, 0, len(records))
		for _, r := range records {
			items = append(items, callFromRecord(r))
		}
		return e.JSON(http.StatusOK, map[string]any{"items": items})
	}
}

// HandleCallEvents streams live call events for the org over SSE. The browser
// connects so an inbound "ringing" event can surface an answer banner instantly.
func HandleCallEvents(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		orgID := access.OrgID

		ch := callhub.DefaultHub.Subscribe(orgID)
		defer callhub.DefaultHub.Unsubscribe(orgID, ch)

		e.Response.Header().Set("Content-Type", "text/event-stream")
		e.Response.Header().Set("Cache-Control", "no-store")
		e.Response.Header().Set("X-Accel-Buffering", "no")
		rc := http.NewResponseController(e.Response)
		rc.SetWriteDeadline(time.Time{})

		ctx := e.Request.Context()
		write := func(name string, ev callhub.CallEvent) error {
			data, err := json.Marshal(ev)
			if err != nil {
				return err
			}
			if _, err := e.Response.Write([]byte("event: " + name + "\ndata: " + string(data) + "\n\n")); err != nil {
				return err
			}
			return rc.Flush()
		}

		for {
			select {
			case <-ctx.Done():
				return nil
			case ev := <-ch:
				name := "update"
				if ev.State == store.CallRinging {
					name = "incoming"
				}
				if err := write(name, ev); err != nil {
					return nil
				}
			}
		}
	}
}

func parseLimit(s string) int {
	if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && n > 0 {
		return n
	}
	return 50
}
