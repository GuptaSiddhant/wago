package store

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// Voice call statuses.
const (
	CallMissed  = "missed"
	CallRinging = "ringing"
	CallActive  = "active"
	CallEnded   = "ended"
	CallFailed  = "failed"
)

// Call directions.
const (
	CallDirectionOutbound = "outbound"
	CallDirectionInbound  = "inbound"
)

// CreateIncomingCall records a new inbound call (ringing) against a conversation.
func CreateIncomingCall(app core.App, conv *core.Record, direction, phone, name string, ts time.Time) (*core.Record, error) {
	col, err := app.FindCollectionByNameOrId("voice_calls")
	if err != nil {
		return nil, err
	}
	rec := core.NewRecord(col)
	rec.Set("org", conv.GetString("org"))
	rec.Set("account", conv.GetString("whatsapp_account"))
	rec.Set("contact", conv.GetString("contact"))
	rec.Set("conversation", conv.Id)
	rec.Set("direction", direction)
	rec.Set("status", CallRinging)
	rec.Set("phone", phone)
	if name != "" {
		rec.Set("peer_name", name)
	}
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("failed to create call record: %w", err)
	}
	return rec, nil
}

// FindOrgCall returns an org-scoped call record by id.
func FindOrgCall(app core.App, orgID, callID string) (*core.Record, error) {
	return FindOrgRecord(app, orgID, "voice_calls", callID)
}

// ListConversationCalls returns the recent calls for a conversation, newest first.
func ListConversationCalls(app core.App, orgID, convID string, limit int) ([]*core.Record, error) {
	if limit <= 0 {
		limit = 50
	}
	return app.FindRecordsByFilter("voice_calls",
		"org = {:org} && conversation = {:conv}", "-created", limit, 0,
		DbxParams(map[string]any{"org": orgID, "conv": convID}))
}

// SetCallStatus transitions a call and records timestamps/duration.
func SetCallStatus(app core.App, callID, status string) error {
	rec, err := app.FindRecordById("voice_calls", callID)
	if err != nil {
		return err
	}
	rec.Set("status", status)
	now := time.Now()
	switch status {
	case CallActive:
		rec.Set("started_at", now)
	case CallEnded, CallFailed, CallMissed:
		rec.Set("ended_at", now)
		if start := rec.GetDateTime("started_at").Time(); !start.IsZero() {
			rec.Set("duration", int(now.Sub(start).Seconds()))
		}
	}
	return app.Save(rec)
}
