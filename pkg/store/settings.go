package store

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase/core"
)

const settingsCollectionName = "app_settings"

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

	return nil
}

// GetSettingsRecord returns the singleton settings record, creating it if it
// does not yet exist.
func GetSettingsRecord(app core.App) (*core.Record, error) {
	rec, err := app.FindRecordById(settingsCollectionName, "default")
	if err == nil {
		return rec, nil
	}

	col, err := app.FindCollectionByNameOrId(settingsCollectionName)
	if err != nil {
		return nil, err
	}
	rec = core.NewRecord(col)
	rec.Set("id", "default")
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
	rec, err := app.FindRecordById(settingsCollectionName, "default")
	if err != nil {
		return "", nil // record not created yet — treat as unset
	}
	return rec.GetString("config"), nil
}
