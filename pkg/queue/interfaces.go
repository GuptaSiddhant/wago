package queue

import (
	"context"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/guptasiddhant/wago/pkg/meta"
	"github.com/guptasiddhant/wago/pkg/store"
)

// MessageSender defines the interface for sending WhatsApp messages. It is
// satisfied by *meta.Client; tests can supply a mock instead.
type MessageSender interface {
	SendText(ctx context.Context, accessToken, phoneNumberID, to, body string) (string, error)
	SendTemplate(ctx context.Context, accessToken, phoneNumberID, to, name, language string, params []map[string]any, header *meta.TemplateHeaderMedia) (string, error)
}

// BroadcastStore defines the interface for broadcast queue operations.
// This allows the worker to be tested with a mock store.
type BroadcastStore interface {
	FindActiveBroadcasts(app core.App) ([]*core.Record, error)
	FindBroadcast(app core.App, broadcastID string) (*core.Record, error)
	ClaimDueRecipients(app core.App, broadcastID string, limit, leaseSeconds int) ([]*core.Record, error)
	MarkRecipientSent(app core.App, rec *core.Record, wamid string) error
	RetryOrFailRecipient(app core.App, rec *core.Record, maxAttempts int, backoffBase time.Duration) (bool, error)
	IncrementBroadcastCounters(app core.App, broadcastID string, sent, failed int) error
	HasOutstandingWork(app core.App, broadcastID string) (bool, error)
	FinalizeBroadcast(app core.App, bc *core.Record) error
	ReleaseExpiredLeases(app core.App, cutoff time.Time) (int64, error)
}

// WorkerDeps holds the dependencies for the broadcast worker. Nil Store/Sender
// fall back to the real implementations; tests inject mocks.
type WorkerDeps struct {
	App    core.App
	Store  BroadcastStore
	Sender MessageSender
	Config Config
}

// NewWorkerWithDeps creates a worker with injected dependencies.
func NewWorkerWithDeps(deps WorkerDeps) *Worker {
	def := DefaultConfig()
	cfg := deps.Config
	if cfg.MessagesPerMinute < 1 {
		cfg.MessagesPerMinute = def.MessagesPerMinute
	}
	if cfg.BatchSize < 1 {
		cfg.BatchSize = def.BatchSize
	}
	if cfg.LeaseSeconds < 1 {
		cfg.LeaseSeconds = def.LeaseSeconds
	}
	if cfg.MaxAttempts < 1 {
		cfg.MaxAttempts = def.MaxAttempts
	}
	if cfg.BackoffBase <= 0 {
		cfg.BackoffBase = def.BackoffBase
	}
	if cfg.LeaseRecoveryInterval <= 0 {
		cfg.LeaseRecoveryInterval = def.LeaseRecoveryInterval
	}

	st := deps.Store
	if st == nil {
		st = &defaultBroadcastStore{}
	}
	sender := deps.Sender
	if sender == nil {
		sender = meta.NewClient()
	}

	return &Worker{
		app:     deps.App,
		store:   st,
		sender:  sender,
		cfg:     cfg,
		limiter: newRateLimiter(cfg.MessagesPerMinute),
	}
}

// defaultBroadcastStore implements BroadcastStore using the store package.
type defaultBroadcastStore struct{}

func (d *defaultBroadcastStore) FindActiveBroadcasts(app core.App) ([]*core.Record, error) {
	return store.FindActiveBroadcasts(app)
}

func (d *defaultBroadcastStore) FindBroadcast(app core.App, broadcastID string) (*core.Record, error) {
	return store.FindBroadcast(app, broadcastID)
}

func (d *defaultBroadcastStore) ClaimDueRecipients(app core.App, broadcastID string, limit, leaseSeconds int) ([]*core.Record, error) {
	return store.ClaimDueRecipients(app, broadcastID, limit, leaseSeconds)
}

func (d *defaultBroadcastStore) MarkRecipientSent(app core.App, rec *core.Record, wamid string) error {
	return store.MarkRecipientSent(app, rec, wamid)
}

func (d *defaultBroadcastStore) RetryOrFailRecipient(app core.App, rec *core.Record, maxAttempts int, backoffBase time.Duration) (bool, error) {
	return store.RetryOrFailRecipient(app, rec, maxAttempts, backoffBase)
}

func (d *defaultBroadcastStore) IncrementBroadcastCounters(app core.App, broadcastID string, sent, failed int) error {
	return store.IncrementBroadcastCounters(app, broadcastID, sent, failed)
}

func (d *defaultBroadcastStore) HasOutstandingWork(app core.App, broadcastID string) (bool, error) {
	return store.HasOutstandingWork(app, broadcastID)
}

func (d *defaultBroadcastStore) FinalizeBroadcast(app core.App, bc *core.Record) error {
	return store.FinalizeBroadcast(app, bc)
}

func (d *defaultBroadcastStore) ReleaseExpiredLeases(app core.App, cutoff time.Time) (int64, error) {
	return store.ReleaseExpiredLeases(app, cutoff)
}