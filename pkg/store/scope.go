package store

import (
	"database/sql"

	"github.com/pocketbase/pocketbase/core"
)

// FindOrgRecord fetches a record by id scoped to a single org. It returns
// sql.ErrNoRows if the record does not exist or belongs to a different org,
// so a caller can never read (or mutate) another organization's data through
// an id-shaped reference. Collection fields that don't carry an "org" column
// (e.g. "users") are not supported here.
func FindOrgRecord(app core.App, orgID, collectionName, id string) (*core.Record, error) {
	rec, err := app.FindRecordById(collectionName, id)
	if err != nil {
		return nil, err
	}
	if rec.GetString("org") != orgID {
		return nil, sql.ErrNoRows
	}
	return rec, nil
}
