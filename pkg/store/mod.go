package store

import (
	"context"
	"log"
	"runtime/debug"
	"sync"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

const appContextKey = "wago.app.ctx"

// AppContext holds the root context cancelled on shutdown and a waitgroup
// for tracking in-flight background work.
type AppContext struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// WithContext returns a new context that inherits from the app root context.
func (ac *AppContext) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, appContextKey, ac)
}

// Root returns the root application context (cancelled on shutdown).
func (ac *AppContext) Root() context.Context {
	return ac.ctx
}

// Cancel cancels the root context.
func (ac *AppContext) Cancel() {
	ac.cancel()
}

// Go launches a background goroutine tracked by the waitgroup. Panics inside
// fn are recovered and logged so one bad delivery can't crash the server;
// cancellation still propagates through the root context.
func (ac *AppContext) Go(fn func(context.Context)) {
	ac.wg.Add(1)
	go func() {
		defer ac.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("background task panicked: %v\n%s", r, debug.Stack())
			}
		}()
		fn(ac.ctx)
	}()
}

// Wait blocks until all tracked goroutines complete.
func (ac *AppContext) Wait() {
	ac.wg.Wait()
}

// GetAppContext retrieves the AppContext from the app's store.
func GetAppContext(app core.App) *AppContext {
	if v := app.Store().Get(appContextKey); v != nil {
		if ac, ok := v.(*AppContext); ok {
			return ac
		}
	}
	return nil
}

// NewAppContext creates a new AppContext with the given context.
func NewAppContext(ctx context.Context) *AppContext {
	ctx, cancel := context.WithCancel(ctx)
	return &AppContext{ctx: ctx, cancel: cancel}
}

// SetAppContext stores the AppContext in the app's store.
func SetAppContext(app core.App, ac *AppContext) {
	app.Store().Set(appContextKey, ac)
}

// GoBackground runs fn as tracked background work when an AppContext is
// registered on app (so shutdown awaits it); otherwise it runs as a detached,
// panic-recovered goroutine. Use it from request handlers that must return to
// the caller promptly while deferring work.
func GoBackground(app core.App, ctx context.Context, fn func(context.Context)) {
	if ac := GetAppContext(app); ac != nil {
		ac.Go(fn)
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("background task panicked: %v\n%s", r, debug.Stack())
			}
		}()
		fn(ctx)
	}()
}

// DbxParams converts a map[string]any into dbx.Params for use with
// the FindRecordsByFilter/FindFirstRecordByFilter filter builders.
func DbxParams(m map[string]any) dbx.Params {
	return dbx.Params(m)
}

// EnsureCollections creates every collection the app relies on using the
// declarative schema system. Falls back to legacy Ensure* functions for
// collections not yet migrated.
func EnsureCollections(app core.App) error {
	// Register all schemas
	RegisterCoreSchemas()

	// Create/update all collections from schemas
	if err := EnsureSchema(app); err != nil {
		return err
	}

	// Resolve relation field CollectionIds
	if err := ResolveRelationFields(app); err != nil {
		return err
	}

	// Legacy migrations that don't fit the schema system
	if err := MigrateLegacyVAPIDKeys(app); err != nil {
		return err
	}

	// users is a PocketBase-owned auth collection, so its optional phone field
	// (used for WhatsApp notifications) is added imperatively.
	if err := EnsureUserPhoneField(app); err != nil {
		return err
	}

	// Enforce org data isolation on existing collections too (the schema system
	// sets rules declaratively, but pre-existing databases created before the
	// schema system may still carry old, looser rules).
	if err := EnforceCollectionSecurity(app); err != nil {
		return err
	}

	return nil
}

func setCollectionRules(app core.App, name string, listView string) error {
	col, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		return nil // collection not (yet) created — nothing to enforce
	}

	col.ListRule = nil
	col.ViewRule = nil
	col.CreateRule = nil
	col.UpdateRule = nil
	col.DeleteRule = nil

	if listView != "" {
		col.ListRule = types.Pointer(listView)
		col.ViewRule = types.Pointer(listView)
	}

	return app.Save(col)
}

// EnforceCollectionSecurity applies the org-isolation access rules to every
// org-scoped collection. org_members lets each user see only their own
// memberships; every other org-scoped collection is locked to superusers and
// the scoped Wago API handlers.
func EnforceCollectionSecurity(app core.App) error {
	for _, name := range []string{
		"orgs",
		"teams",
		"whatsapp_accounts",
		"contacts",
		"conversations",
		"messages",
		"invites",
		"message_templates",
		"broadcasts",
		"broadcast_recipients",
		"voice_calls",
	} {
		if err := setCollectionRules(app, name, ""); err != nil {
			return err
		}
	}
	// Users may only see their own account record through the raw API; this
	// prevents cross-user email enumeration while keeping auth flows intact
	// (all password/management flows run server-side, bypassing these rules).
	if err := setCollectionRules(app, "users", "id = @request.auth.id"); err != nil {
		return err
	}
	// org_members is the one place a rule can enforce org membership directly.
	return setCollectionRules(app, "org_members", "user = @request.auth.id")
}
