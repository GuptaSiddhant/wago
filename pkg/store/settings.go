package store

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase/core"
)

const (
	settingsCollectionName = "app_settings"
	// settingsRecordID is the fixed id of the singleton settings record. It
	// must match PocketBase's id rules exactly: 15 lowercase alphanumeric
	// characters (min/max 15, pattern ^[a-z0-9]+$), so the record can be
	// created with a stable, well-known id.
	settingsRecordID = "defaultsettings"
)

// EnsureSettingsCollection creates the singleton app_settings collection that
// holds the runtime-editable Wago configuration. It is a single-record base
// collection (fixed id "default") locked down to superusers only, so the config
// can only be read/updated through the Wago API.
func EnsureSettingsCollection(app core.App) error {
	if _, err := app.FindCollectionByNameOrId(settingsCollectionName); err != nil {
		col := core.NewBaseCollection(settingsCollectionName)
		col.ListRule = nil
		col.ViewRule = nil
		col.CreateRule = nil
		col.UpdateRule = nil
		col.DeleteRule = nil

		col.Fields.Add(&core.JSONField{Name: "config"})

		if err := app.Save(col); err != nil {
			return fmt.Errorf("failed to auto-create app_settings collection: %w", err)
		}
		log.Println("Auto-created 'app_settings' collection")
	}

	// Add the Web Push VAPID keypair fields, including to pre-existing
	// databases (the collection may already exist from an earlier boot).
	return ensureFields(app, settingsCollectionName,
		&core.TextField{Name: "vapid_public_key"},
		&core.TextField{Name: "vapid_private_key"})
}

// GetVAPIDKeysFromSettings returns the VAPID keypair stored on the singleton
// settings record, or empty strings when not yet generated.
func GetVAPIDKeysFromSettings(app core.App) (publicKey, privateKey string, err error) {
	rec, err := GetSettingsRecord(app)
	if err != nil {
		return "", "", err
	}
	return rec.GetString("vapid_public_key"), rec.GetString("vapid_private_key"), nil
}

// SaveVAPIDKeysToSettings stores the Web Push VAPID keypair on the singleton
// settings record.
func SaveVAPIDKeysToSettings(app core.App, publicKey, privateKey string) error {
	rec, err := GetSettingsRecord(app)
	if err != nil {
		return err
	}
	rec.Set("vapid_public_key", publicKey)
	rec.Set("vapid_private_key", privateKey)
	return app.Save(rec)
}

// GetSettingsRecord returns the singleton settings record, creating it if it
// does not yet exist. A legacy record with the id "default" (which pre-dates
// PocketBase's 15-character id minimum) is migrated to the current id.
func GetSettingsRecord(app core.App) (*core.Record, error) {
	rec, err := app.FindRecordById(settingsCollectionName, settingsRecordID)
	if err == nil {
		return rec, nil
	}

	// Migrate a record created before the id change.
	if legacy, lerr := app.FindRecordById(settingsCollectionName, "default"); lerr == nil {
		legacy.Set("id", settingsRecordID)
		if serr := app.Save(legacy); serr != nil {
			return nil, fmt.Errorf("failed to migrate app_settings record id: %w", serr)
		}
		return legacy, nil
	}

	col, err := app.FindCollectionByNameOrId(settingsCollectionName)
	if err != nil {
		return nil, err
	}
	rec = core.NewRecord(col)
	rec.Set("id", settingsRecordID)
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("failed to create app_settings record: %w", err)
	}
	return rec, nil
}

// SaveSettingsConfig persists the given JSON config blob into the singleton
// settings record.
func SaveSettingsConfig(app core.App, configJSON string) error {
	rec, err := GetSettingsRecord(app)
	if err != nil {
		return err
	}
	rec.Set("config", configJSON)
	if err := app.Save(rec); err != nil {
		return fmt.Errorf("failed to save app settings: %w", err)
	}
	return nil
}

// GetSettingsConfig returns the stored JSON config blob, or "" when it is unset.
func GetSettingsConfig(app core.App) (string, error) {
	rec, err := app.FindRecordById(settingsCollectionName, settingsRecordID)
	if err != nil {
		return "", nil // record not created yet — treat as unset
	}
	return rec.GetString("config"), nil
}
