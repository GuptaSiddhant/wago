package queue

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/guptasiddhant/wago/pkg/meta"
	"github.com/guptasiddhant/wago/pkg/store"
)

// Config tunes how aggressively the broadcast worker sends messages.
type Config struct {
	MessagesPerMinute     int
	BatchSize             int
	LeaseSeconds          int
	MaxAttempts           int
	BackoffBase           time.Duration
	LeaseRecoveryInterval time.Duration // how often expired leases are swept
}

// DefaultConfig returns a sane single-instance configuration.
func DefaultConfig() Config {
	return Config{
		MessagesPerMinute:     60,
		BatchSize:             10,
		LeaseSeconds:          300,
		MaxAttempts:           3,
		BackoffBase:           30 * time.Second,
		LeaseRecoveryInterval: time.Minute,
	}
}

// Worker drains queued broadcast recipients using a SQLite-backed lease queue.
// A single in-process goroutine claims small batches, sends them under a global
// token-bucket rate limit, and records per-recipient results. Expired leases
// (crashed workers) are swept back into the queue, so the only job of a cron is
// to wake the worker after a restart — it never double-sends, even if multiple
// instances are running.
type Worker struct {
	app     core.App
	client  *meta.Client
	cfg     Config
	limiter *rateLimiter
}

// NewWorker builds a worker. Zero/negative config values fall back to defaults.
func NewWorker(app core.App, cfg Config) *Worker {
	def := DefaultConfig()
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
	return &Worker{
		app:     app,
		client:  meta.NewClient(),
		cfg:     cfg,
		limiter: newRateLimiter(cfg.MessagesPerMinute),
	}
}

// Run processes active broadcasts until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	log.Printf("queue: broadcast worker started (%d/min, batch %d, lease %ds)",
		w.cfg.MessagesPerMinute, w.cfg.BatchSize, w.cfg.LeaseSeconds)

	lastLeaseSweep := time.Time{}
	tick := 250 * time.Millisecond

	for {
		if time.Since(lastLeaseSweep) >= w.cfg.LeaseRecoveryInterval {
			w.sweepExpiredLeases()
			lastLeaseSweep = time.Now()
		}

		w.ProcessTick(ctx)

		select {
		case <-ctx.Done():
			log.Println("queue: broadcast worker stopped")
			return
		case <-time.After(tick):
		}
	}
}

// sweepExpiredLeases returns recipients stuck in "sending" past their lease
// back to "queued" so they can be retried after a crash/restart.
func (w *Worker) sweepExpiredLeases() {
	released, err := store.ReleaseExpiredLeases(w.app, time.Now())
	if err != nil {
		log.Printf("queue: lease sweep failed: %v", err)
		return
	}
	if released > 0 {
		log.Printf("queue: released %d expired recipient leases", released)
	}
}

// ProcessTick does one pass over all active broadcasts: claim a batch, send it,
// record results, and finalize broadcasts with no work left. It is safe to call
// from tests and keeps the whole worker single-threaded per database.
func (w *Worker) ProcessTick(ctx context.Context) {
	broadcasts, err := store.FindActiveBroadcasts(w.app)
	if err != nil {
		log.Printf("queue: list active broadcasts failed: %v", err)
		return
	}
	for _, bc := range broadcasts {
		if err := w.processBroadcast(ctx, bc); err != nil {
			log.Printf("queue: broadcast %s processing failed: %v", bc.Id, err)
		}
	}
}

func (w *Worker) processBroadcast(ctx context.Context, bc *core.Record) error {
	id := bc.Id

	fresh, err := store.FindBroadcast(w.app, id)
	if err != nil {
		return err
	}
	if fresh.GetString("status") == store.BroadcastCancelled {
		return nil
	}

	if fresh.GetString("status") == store.BroadcastQueued {
		fresh.Set("status", store.BroadcastRunning)
		fresh.Set("started_at", time.Now())
		if err := w.app.Save(fresh); err != nil {
			return err
		}
	}

	batch := w.cfg.BatchSize
	if perBc := fresh.GetInt("batch_size"); perBc > 0 {
		batch = perBc
	}

	claimed, err := store.ClaimDueRecipients(w.app, id, batch, w.cfg.LeaseSeconds)
	if err != nil {
		return err
	}

	failed := 0
	for _, rec := range claimed {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		w.limiter.Wait(ctx)

		wamid, sendErr := w.sendTemplate(ctx, fresh, rec)
		if sendErr != nil {
			isFinal, markErr := store.RetryOrFailRecipient(w.app, rec, w.cfg.MaxAttempts, w.cfg.BackoffBase)
			if markErr != nil {
				log.Printf("queue: broadcast %s recipient %s mark failed: %v", id, rec.GetString("phone"), markErr)
				continue
			}
			if isFinal {
				failed++
			}
			log.Printf("queue: broadcast %s send to %s failed (attempt %d): %v",
				id, rec.GetString("phone"), rec.GetInt("attempts"), sendErr)
			continue
		}
		if err := store.MarkRecipientSent(w.app, rec, wamid); err != nil {
			log.Printf("queue: broadcast %s recipient %s mark sent failed: %v", id, rec.GetString("phone"), err)
			continue
		}
		fresh.Set("sent_count", fresh.GetInt("sent_count")+1)
	}

	if failed > 0 {
		fresh.Set("failed_count", fresh.GetInt("failed_count")+failed)
	}
	if err := w.app.Save(fresh); err != nil {
		return err
	}

	// Finalize once nothing is left queued or mid-flight.
	outstanding, err := store.HasOutstandingWork(w.app, id)
	if err != nil {
		return err
	}
	if !outstanding {
		latest, err := store.FindBroadcast(w.app, id)
		if err != nil {
			return err
		}
		if err := store.FinalizeBroadcast(w.app, latest); err != nil {
			return err
		}
		log.Printf("queue: broadcast %s finalized (%d sent, %d failed)",
			id, latest.GetInt("sent_count"), latest.GetInt("failed_count"))
	}
	return nil
}

// sendTemplate sends one template message to a recipient.
func (w *Worker) sendTemplate(ctx context.Context, bc *core.Record, recipient *core.Record) (string, error) {
	account, err := w.app.FindRecordById("whatsapp_accounts", bc.GetString("account"))
	if err != nil {
		return "", err
	}
	tmpl, err := w.app.FindRecordById("message_templates", bc.GetString("template"))
	if err != nil {
		return "", err
	}

	params := decodeBroadcastParams(bc)
	return w.client.SendTemplate(ctx,
		account.GetString("access_token"),
		account.GetString("phone_number_id"),
		recipient.GetString("phone"),
		tmpl.GetString("name"),
		tmpl.GetString("language"),
		params)
}

// decodeBroadcastParams reads the broadcast params JSON field, which PocketBase
// may expose as a raw message or a typed slice depending on storage.
func decodeBroadcastParams(bc *core.Record) []map[string]any {
	var params []map[string]any
	raw := bc.Get("params")
	if raw == nil {
		return params
	}

	var data []byte
	switch v := raw.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	case json.RawMessage:
		data = v
	default:
		if typed, ok := v.([]map[string]any); ok {
			return typed
		}
		return params
	}

	_ = json.Unmarshal(data, &params)
	return params
}

// rateLimiter is a token bucket that paces sends to `perMinute` while allowing
// bursts of up to one second's worth of tokens (that's the "batching": we grab
// a batch fast, but the sustained rate stays capped).
type rateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	capacity   float64
	ratePerSec float64
	last       time.Time
}

func newRateLimiter(perMinute int) *rateLimiter {
	rate := float64(perMinute) / 60.0
	return &rateLimiter{tokens: rate, capacity: rate, ratePerSec: rate, last: time.Now()}
}

// Wait blocks until a token is available or the context is cancelled.
func (l *rateLimiter) Wait(ctx context.Context) {
	for {
		if l.allow() {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (l *rateLimiter) allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(l.last).Seconds()
	l.last = now
	l.tokens = math.Min(l.capacity, l.tokens+elapsed*l.ratePerSec)
	if l.tokens >= 1 {
		l.tokens--
		return true
	}
	return false
}
