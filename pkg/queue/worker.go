package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/guptasiddhant/wago/pkg/store"
)

// Worker drains queued broadcast recipients using a SQLite-backed lease queue.
// A single in-process goroutine claims small batches, sends them under a global
// token-bucket rate limit, and records per-recipient results. Expired leases
// (crashed workers) are swept back into the queue, so the only job of a cron is
// to wake the worker after a restart — it never double-sends, even if multiple
// instances are running.
type Worker struct {
	app    core.App
	store  BroadcastStore
	sender MessageSender
	cfg    Config
	limiter *rateLimiter
}

// logf emits a structured log through PocketBase's slog logger when an app is
// available, falling back to the standard logger (e.g. in unit tests).
func (w *Worker) logf(level slog.Level, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if w.app != nil {
		w.app.Logger().Log(context.Background(), level, msg,
			slog.String("component", "queue"))
		return
	}
	slog.Log(context.Background(), level, msg, slog.String("component", "queue"))
}

// NewWorker builds a worker. Zero/negative config values fall back to defaults.
func NewWorker(app core.App, cfg Config) *Worker {
	return NewWorkerWithDeps(WorkerDeps{
		App:    app,
		Config: cfg,
	})
}

// Run processes active broadcasts until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	w.logf(slog.LevelInfo, "broadcast worker started (%d/min, batch %d, lease %ds)",
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
			w.logf(slog.LevelInfo, "broadcast worker stopped")
			return
		case <-time.After(tick):
		}
	}
}

// sweepExpiredLeases returns recipients stuck in "sending" past their lease
// back to "queued" so they can be retried after a crash/restart.
func (w *Worker) sweepExpiredLeases() {
	released, err := w.store.ReleaseExpiredLeases(w.app, time.Now())
	if err != nil {
		w.logf(slog.LevelWarn, "lease sweep failed: %v", err)
		return
	}
	if released > 0 {
		w.logf(slog.LevelInfo, "released %d expired recipient leases", released)
	}
}

// ProcessTick does one pass over all active broadcasts: claim a batch, send it,
// record results, and finalize broadcasts with no work left. It is safe to call
// from tests and keeps the whole worker single-threaded per database.
func (w *Worker) ProcessTick(ctx context.Context) {
	broadcasts, err := w.store.FindActiveBroadcasts(w.app)
	if err != nil {
		w.logf(slog.LevelWarn, "list active broadcasts failed: %v", err)
		return
	}
	for _, bc := range broadcasts {
		if err := w.processBroadcast(ctx, bc); err != nil {
			w.logf(slog.LevelError, "broadcast %s processing failed: %v", bc.Id, err)
		}
	}
}

func (w *Worker) processBroadcast(ctx context.Context, bc *core.Record) error {
	id := bc.Id

	fresh, err := w.store.FindBroadcast(w.app, id)
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

	claimed, err := w.store.ClaimDueRecipients(w.app, id, batch, w.cfg.LeaseSeconds)
	if err != nil {
		return err
	}

	failed := 0
	sent := 0
	for _, rec := range claimed {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		w.limiter.Wait(ctx)

		wamid, sendErr := w.sendTemplate(ctx, fresh, rec)
		if sendErr != nil {
			isFinal, markErr := w.store.RetryOrFailRecipient(w.app, rec, w.cfg.MaxAttempts, w.cfg.BackoffBase)
			if markErr != nil {
				w.logf(slog.LevelError, "broadcast %s recipient %s mark failed: %v",
					id, rec.GetString("phone"), markErr)
				continue
			}
			if isFinal {
				failed++
			}
			w.logf(slog.LevelWarn, "broadcast %s send to %s failed (attempt %d): %v",
				id, rec.GetString("phone"), rec.GetInt("attempts"), sendErr)
			continue
		}
		if err := w.store.MarkRecipientSent(w.app, rec, wamid); err != nil {
			w.logf(slog.LevelError, "broadcast %s recipient %s mark sent failed: %v",
				id, rec.GetString("phone"), err)
			continue
		}
		sent++
	}

	// Persist counters atomically so concurrent workers never lose updates.
	if sent > 0 || failed > 0 {
		if err := w.store.IncrementBroadcastCounters(w.app, id, sent, failed); err != nil {
			return err
		}
	}

	// Finalize once nothing is left queued or mid-flight.
	outstanding, err := w.store.HasOutstandingWork(w.app, id)
	if err != nil {
		return err
	}
	if !outstanding {
		latest, err := w.store.FindBroadcast(w.app, id)
		if err != nil {
			return err
		}
		if err := w.store.FinalizeBroadcast(w.app, latest); err != nil {
			return err
		}
		w.logf(slog.LevelInfo, "broadcast %s finalized (%d sent, %d failed)",
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

	// A per-broadcast media override wins over the template's own header media.
	var header *TemplateHeaderMedia
	if h := store.TemplateHeaderMedia(tmpl); h != nil {
		header = &TemplateHeaderMedia{Kind: h.Kind, MediaID: h.MediaID}
	}
	if kind := bc.GetString("header_media_type"); kind != "" {
		if id := bc.GetString("header_media_id"); id != "" {
			header = &TemplateHeaderMedia{Kind: strings.ToLower(kind), MediaID: id}
		}
	}

	return w.sender.SendTemplate(ctx,
		account.GetString("access_token"),
		account.GetString("phone_number_id"),
		recipient.GetString("phone"),
		tmpl.GetString("name"),
		tmpl.GetString("language"),
		params,
		header)
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
