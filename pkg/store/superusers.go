package store

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

func EnsureSuperuser(app core.App, email, password string) error {
	superusersCol, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		return err
	}
	count, err := app.CountRecords(superusersCol)
	if err != nil || count > 0 {
		return nil
	}

	if email == "" {
		email = "admin@wago.local"
	}
	if password == "" {
		password = "Password123"
	}

	record := core.NewRecord(superusersCol)
	record.Set("email", email)
	record.Set("password", password)
	record.Set("passwordConfirm", password)

	err = app.Save(record)
	if err != nil {
		fmt.Printf("Created new superuser/admin: '%s'", email)
	}

	return err
}
