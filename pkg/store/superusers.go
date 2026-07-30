package store

import (
	"os"

	"github.com/pocketbase/pocketbase/core"
)

func EnsureSuperuser(app core.App) error {
	superusersCol, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		return err
	}
	count, err := app.CountRecords(superusersCol)
	if err != nil || count > 0 {
		return nil
	}

	email := os.Getenv("ADMIN_EMAIL")
	if email == "" {
		email = "admin@wago.local"
	}
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = "Password123"
	}

	record := core.NewRecord(superusersCol)
	record.Set("email", email)
	record.Set("password", password)
	record.Set("passwordConfirm", password)

	return app.Save(record)
}
