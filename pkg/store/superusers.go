package store

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

// EnsureSuperuser creates the initial admin account if none exists yet. It is
// safe to call on every boot and returns nil when an admin already exists.
func EnsureSuperuser(app core.App, email, password string) error {
	if (password == "") || (email == "") {
		return fmt.Errorf("superuser email and password must be provided to create superuser")
	}

	superusersCol, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		return err
	}
	count, err := app.CountRecords(superusersCol)
	if err != nil || count > 0 {
		return nil
	}

	record := core.NewRecord(superusersCol)
	record.Set("email", email)
	record.Set("password", password)
	record.Set("passwordConfirm", password)

	err = app.Save(record)
	if err != nil {
		fmt.Printf("Failed to create superuser/admin '%s': %v", email, err)
	}

	return err
}
