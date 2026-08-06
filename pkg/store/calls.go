package store

import (
	"fmt"
	"log"
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

// EnsureVoiceCallsCollection creates the voice_calls collection that records
// every in/outbound call against a conversation.
func EnsureVoiceCallsCollection(app core.App) error {
	orgsCol, err := app.FindCollectionByNameOrId("orgs")
	if err != nil {
		return fmt.Errorf("orgs collection not found: %w", err)
	}
	accountsCol, err := app.FindCollectionByNameOrId("whatsapp_accounts")
	if err != nil {
		return fmt.Errorf("whatsapp_accounts collection not found: %w", err)
	}
	contactsCol, err := app.FindCollectionByNameOrId("contacts")
	if err != nil {
		return fmt.Errorf("contacts collection not found: %w", err)
	}
	convsCol, err := app.FindCollectionByNameOrId("conversations")
	if err != nil {
		return fmt.Errorf("conversations collection not found: %w", err)
	}

	if _, err := app.FindCollectionByNameOrId("voice_calls"); err != nil {
		col := core.NewBaseCollection("voice_calls")
		col.ListRule = nil
		col.ViewRule = nil
		col.CreateRule = nil
		col.UpdateRule = nil
		col.DeleteRule = nil

		col.Fields.Add(
			&core.RelationField{Name: "org", CollectionId: orgsCol.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
			&core.RelationField{Name: "account", CollectionId: accountsCol.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
			&core.RelationField{Name: "contact", CollectionId: contactsCol.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
			&core.RelationField{Name: "conversation", CollectionId: convsCol.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
			&core.TextField{Name: "direction", Required: true},
			&core.TextField{Name: "status", Required: true},
			&core.TextField{Name: "phone", Required: true},
			&core.TextField{Name: "peer_name"},
			&core.DateField{Name: "started_at"},
			&core.DateField{Name: "answered_at"},
			&core.DateField{Name: "ended_at"},
			&core.NumberField{Name: "duration"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		col.AddIndex("idx_voice_calls_conversation", false, "conversation", "")

		if err := app.Save(col); err != nil {
			return fmt.Errorf("failed to auto-create voice_calls collection: %w", err)
		}
		log.Println("Auto-created 'voice_calls' collection")
	}

	return nil
}

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
