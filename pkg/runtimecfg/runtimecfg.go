package runtimecfg

import (
	"encoding/json"
	"log"

	"github.com/guptasiddhant/wago/pkg/store"
	"github.com/guptasiddhant/wago/pkg/utils"

	"github.com/pocketbase/pocketbase/core"
)

// Manager provides the effective Wago configuration. On boot the env-seeded
// config is persisted into the app_settings collection so a superadmin can edit
// it at runtime. Load reads the latest persisted values on every call so changes
// made through the API are reflected immediately (no restart required).
//
// AdminPassword and AdminEmail are never persisted and always come from the
// env-seeded config.
type Manager struct {
	env *utils.AppConfig
}

// New builds a Manager whose env-seeded config is the fallback for any values
// that have not yet been persisted.
func New(env *utils.AppConfig) *Manager {
	return &Manager{env: env}
}

// Env returns the env-seeded config. It is read-only; callers must not mutate
// the returned pointer.
func (m *Manager) Env() *utils.AppConfig {
	return m.env
}

// Seed persists the env-seeded config into the DB if no config has been saved
// yet. Safe to call on every boot; it only writes when the DB is empty.
func (m *Manager) Seed(app core.App) error {
	existing, err := store.GetSettingsConfig(app)
	if err != nil {
		return err
	}
	if existing != "" {
		return nil // already configured
	}

	// Marshal the editable portion only (admin fields are json:"-").
	b, err := json.Marshal(m.env)
	if err != nil {
		return err
	}
	log.Println("Seeding app settings from environment")
	return store.SaveSettingsConfig(app, string(b))
}

// Load returns the effective config: the persisted values, falling back to the
// env-seeded config for anything not (yet) stored. Admin credentials always come
// from the env-seeded config.
func (m *Manager) Load(app core.App) (*utils.AppConfig, error) {
	cfg := *m.env // copy so callers can't mutate the shared env config

	stored, err := store.GetSettingsConfig(app)
	if err != nil {
		return nil, err
	}
	if stored == "" {
		return &cfg, nil
	}

	if err := json.Unmarshal([]byte(stored), &cfg); err != nil {
		log.Printf("runtimecfg: ignoring malformed persisted config: %v", err)
		return &cfg, nil
	}

	return &cfg, nil
}

// Save persists the editable fields of cfg into the DB. Admin fields are
// ignored (they are json:"-").
func (m *Manager) Save(app core.App, cfg *utils.AppConfig) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return store.SaveSettingsConfig(app, string(b))
}
