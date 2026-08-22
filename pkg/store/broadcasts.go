package store

import (
	"fmt"
	"log"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Broadcast statuses.
const (
	BroadcastQueued    = "queued"
	BroadcastRunning   = "running"
	BroadcastCompleted = "completed"
	BroadcastFailed    = "failed"
	BroadcastCancelled = "cancelled"
)

// Recipient statuses.
const (
	RecipientQueued  = "queued"
	RecipientSending = "sending"
	RecipientSent    = "sent"
	RecipientFailed  = "failed"
)

// RecipientSnapshot is a contact row frozen at broadcast creation time.
type RecipientSnapshot struct {
	ContactID string
	Phone     string
	Name      string
}

// sqlTime renders a time in the storage format PocketBase uses for DateFields so
// raw SQL writes and comparisons stay consistent with PB-read values.
func sqlTime(t time.Time) string {
	return t.UTC().Format(types.DefaultDateLayout)
}

// sqlZeroTime is the sentinel value written to DateField columns when there is
// no value (PocketBase stores NOT NULL date columns and uses this as "empty").
const sqlZeroTime = "0001-01-01 00:00:00.000Z"

// CreateBroadcast atomically creates a broadcast header plus one queued
func ensureFields(app core.App, colName string, fields ...core.Field) error {
	col, err := app.FindCollectionByNameOrId(colName)
	if err != nil {
		return err
	}
	var toAdd []core.Field
	for _, f := range fields {
		if col.Fields.GetByName(f.GetName()) == nil {
			toAdd = append(toAdd, f)
		}
	}
	if len(toAdd) == 0 {
		return nil
	}
	col.Fields.Add(toAdd...)
	if err := app.Save(col); err != nil {
		return fmt.Errorf("failed to save %s schema: %w", colName, err)
	}
	log.Printf("Added fields to '%s': %v", colName, func() []string {
		names := make([]string, 0, len(toAdd))
		for _, f := range toAdd {
			names = append(names, f.GetName())
		}
		return names
	}())
	return nil
}

// CreateBroadcast atomically creates a broadcast header plus one queued
// recipient row per snapshot and returns the broadcast record. headerType,
// headerID and headerName are an optional per-broadcast media override for the
// template's header (empty when the template's own header media is used).
func CreateBroadcast(app core.App, orgID, accountID, templateID, name, createdBy string, params any, ratePerMinute, batchSize int, recipients []RecipientSnapshot, headerType, headerID, headerName string) (*core.Record, error) {
	var bc *core.Record

	err := app.RunInTransaction(func(txApp core.App) error {
		col, err := txApp.FindCollectionByNameOrId("broadcasts")
		if err != nil {
			return err
		}
		bc = core.NewRecord(col)
		bc.Set("org", orgID)
		bc.Set("account", accountID)
		bc.Set("template", templateID)
		bc.Set("created_by", createdBy)
		bc.Set("name", name)
		bc.Set("status", BroadcastQueued)
		bc.Set("params", params)
		bc.Set("header_media_type", headerType)
		bc.Set("header_media_id", headerID)
		bc.Set("header_media_name", headerName)
		bc.Set("rate_per_minute", ratePerMinute)
		bc.Set("batch_size", batchSize)
		bc.Set("recipient_count", len(recipients))
		bc.Set("sent_count", 0)
		bc.Set("failed_count", 0)
		if err := txApp.Save(bc); err != nil {
			return fmt.Errorf("failed to create broadcast: %w", err)
		}

		recipientsCol, err := txApp.FindCollectionByNameOrId("broadcast_recipients")
		if err != nil {
			return err
		}
		for _, snap := range recipients {
			rec := core.NewRecord(recipientsCol)
			rec.Set("org", orgID)
			rec.Set("broadcast", bc.Id)
			rec.Set("contact", snap.ContactID)
			rec.Set("phone", snap.Phone)
			rec.Set("name", snap.Name)
			rec.Set("status", RecipientQueued)
			if err := txApp.Save(rec); err != nil {
				return fmt.Errorf("failed to snapshot recipient %s: %w", snap.Phone, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return bc, nil
}

// FindOrgBroadcast returns an org-scoped broadcast by id.
func FindOrgBroadcast(app core.App, orgID, id string) (*core.Record, error) {
	return FindOrgRecord(app, orgID, "broadcasts", id)
}

// FindBroadcast returns a broadcast by record id regardless of org.
func FindBroadcast(app core.App, broadcastID string) (*core.Record, error) {
	return app.FindRecordById("broadcasts", broadcastID)
}

// FindActiveBroadcasts lists broadcasts that still have work to do.
func FindActiveBroadcasts(app core.App) ([]*core.Record, error) {
	return app.FindRecordsByFilter("broadcasts",
		"status = {:queued} || status = {:running}",
		"created", 200, 0,
		DbxParams(map[string]any{"queued": BroadcastQueued, "running": BroadcastRunning}))
}

// ClaimDueRecipients atomically claims up to limit eligible recipients for a
// broadcast, marking them "sending" with a lease until now+lease. The claim is
// an optimistic UPDATE guarded by `status = queued`, so concurrent workers can
// never double-claim a recipient. Returns the claimed records.
func ClaimDueRecipients(app core.App, broadcastID string, limit, leaseSeconds int) ([]*core.Record, error) {
	var claimed []*core.Record

	err := app.RunInTransaction(func(txApp core.App) error {
		now := time.Now()
		candidates, err := txApp.FindRecordsByFilter("broadcast_recipients",
			"broadcast = {:broadcast} && status = {:queued} && (next_attempt_at = null || next_attempt_at <= {:now})",
			"created", limit, 0,
			DbxParams(map[string]any{
				"broadcast": broadcastID,
				"queued":    RecipientQueued,
				"now":       sqlTime(now),
			}))
		if err != nil {
			return err
		}

		leaseUntil := now.Add(time.Duration(leaseSeconds) * time.Second)
		leaseSQL := sqlTime(leaseUntil)
		nowSQL := sqlTime(now)
		for _, rec := range candidates {
			result, err := txApp.DB().NewQuery(
				`UPDATE broadcast_recipients
				 SET status = {:sending},
				     attempts = attempts + 1,
				     lease_until = {:lease_until},
				     updated = {:now}
				 WHERE id = {:id} AND status = {:queued}`).
				Bind(dbx.Params{
					"sending":     RecipientSending,
					"lease_until": leaseSQL,
					"now":         nowSQL,
					"id":          rec.Id,
					"queued":      RecipientQueued,
				}).Execute()
			if err != nil {
				return err
			}
			affected, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if affected != 1 {
				// Another worker grabbed it first — skip.
				continue
			}
			rec.Set("status", RecipientSending)
			rec.Set("attempts", rec.GetInt("attempts")+1)
			rec.Set("lease_until", leaseUntil)
			claimed = append(claimed, rec)
		}
		return nil
	})

	return claimed, err
}

// ReleaseExpiredLeases returns every recipient whose lease has lapsed (a
// crashed/restarted worker) back to "queued" so another worker can pick them
// up again. Purely a safety net; active workers hold short leases.
func ReleaseExpiredLeases(app core.App, cutoff time.Time) (int64, error) {
	result, err := app.DB().NewQuery(
		`UPDATE broadcast_recipients
		 SET status = {:queued}, lease_until = {:zero}, updated = {:now}
		 WHERE status = {:sending} AND lease_until > {:zero} AND lease_until <= {:cutoff}`).
		Bind(dbx.Params{
			"queued":  RecipientQueued,
			"zero":    sqlZeroTime,
			"now":     sqlTime(time.Now()),
			"sending": RecipientSending,
			"cutoff":  sqlTime(cutoff),
		}).Execute()
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return affected, nil
}

// MarkRecipientSent records a successful delivery.
func MarkRecipientSent(app core.App, rec *core.Record, wamid string) error {
	rec.Set("status", RecipientSent)
	rec.Set("wamid", wamid)
	rec.Set("error", "")
	rec.Set("lease_until", nil)
	rec.Set("sent_at", time.Now())
	return app.Save(rec)
}

// RetryOrFailRecipient releases a failed claim. If attempts are exhausted the
// recipient is marked failed and true is returned; otherwise it is requeued
// with a per-recipient exponential backoff and false is returned.
func RetryOrFailRecipient(app core.App, rec *core.Record, maxAttempts int, backoffBase time.Duration) (bool, error) {
	attempts := rec.GetInt("attempts")
	rec.Set("lease_until", nil)

	if attempts >= maxAttempts {
		rec.Set("status", RecipientFailed)
		rec.Set("error", fmt.Sprintf("failed after %d attempts", attempts))
		if err := app.Save(rec); err != nil {
			return true, err
		}
		return true, nil
	}

	rec.Set("status", RecipientQueued)
	rec.Set("next_attempt_at", time.Now().Add(backoffBase*time.Duration(1<<uint(attempts-1))))
	if err := app.Save(rec); err != nil {
		return false, err
	}
	return false, nil
}

// HasOutstandingWork reports whether a broadcast still has queued recipients or
// recipients being actively sent (with a live lease).
func HasOutstandingWork(app core.App, broadcastID string) (bool, error) {
	queued, err := app.CountRecords("broadcast_recipients",
		dbx.And(dbx.HashExp{"broadcast": broadcastID, "status": RecipientQueued}))
	if err != nil {
		return false, err
	}
	sending, err := app.CountRecords("broadcast_recipients",
		dbx.And(dbx.HashExp{"broadcast": broadcastID, "status": RecipientSending}))
	if err != nil {
		return false, err
	}
	return queued+sending > 0, nil
}

// IncrementBroadcastCounters atomically bumps a broadcast's sent/failed counters
// in the database using a single UPDATE. This avoids the lost-update that would
// occur if multiple worker instances read-modify-write the same row in memory.
func IncrementBroadcastCounters(app core.App, broadcastID string, sent, failed int) error {
	_, err := app.DB().NewQuery(
		`UPDATE broadcasts
		 SET sent_count = sent_count + {:sent}, failed_count = failed_count + {:failed},
		     updated = {:now}
		 WHERE id = {:id}`).
		Bind(dbx.Params{
			"sent":   sent,
			"failed": failed,
			"now":    sqlTime(time.Now()),
			"id":     broadcastID,
		}).Execute()
	return err
}

// FinalizeBroadcast marks a broadcast finished once its queue is drained.
func FinalizeBroadcast(app core.App, bc *core.Record) error {
	status := BroadcastCompleted
	if bc.GetInt("failed_count") > 0 && bc.GetInt("sent_count") == 0 {
		status = BroadcastFailed
	}
	bc.Set("status", status)
	bc.Set("finished_at", time.Now())
	return app.Save(bc)
}

// CountBroadcastRecipients returns the number of rows per status for a broadcast.
func CountBroadcastRecipients(app core.App, broadcastID string) (map[string]int, error) {
	counts := map[string]int{}
	for _, status := range []string{RecipientQueued, RecipientSending, RecipientSent, RecipientFailed} {
		total, err := app.CountRecords("broadcast_recipients",
			dbx.And(dbx.HashExp{"broadcast": broadcastID, "status": status}))
		if err != nil {
			return nil, err
		}
		counts[status] = int(total)
	}
	return counts, nil
}

// CountBroadcastRecipientsBatch returns recipient counts per status for many
// broadcasts in a single grouped query. The result is keyed by broadcast id,
// then by status; broadcasts with no rows are absent from the result.
func CountBroadcastRecipientsBatch(app core.App, broadcastIDs []string) (map[string]map[string]int64, error) {
	out := make(map[string]map[string]int64, len(broadcastIDs))
	if len(broadcastIDs) == 0 {
		return out, nil
	}

	ids := make([]any, len(broadcastIDs))
	for i, id := range broadcastIDs {
		ids[i] = id
	}

	rows := []struct {
		Broadcast string `db:"broadcast"`
		Status    string `db:"status"`
		Total     int64  `db:"total"`
	}{}
	err := app.DB().
		Select("broadcast", "status", "COUNT(*) AS total").
		From("broadcast_recipients").
		Where(dbx.In("broadcast", ids...)).
		GroupBy("broadcast", "status").
		All(&rows)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		if out[row.Broadcast] == nil {
			out[row.Broadcast] = map[string]int64{}
		}
		out[row.Broadcast][row.Status] = row.Total
	}
	return out, nil
}

// ListBroadcastRecipients returns recipient rows (used for progress/detail).
func ListBroadcastRecipients(app core.App, broadcastID string, limit int) ([]*core.Record, error) {
	return app.FindRecordsByFilter("broadcast_recipients",
		"broadcast = {:broadcast}", "-created", limit, 0,
		DbxParams(map[string]any{"broadcast": broadcastID}))
}
