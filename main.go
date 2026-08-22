package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/guptasiddhant/wago/pkg/aichat"
	"github.com/guptasiddhant/wago/pkg/api"
	"github.com/guptasiddhant/wago/pkg/notifications"
	"github.com/guptasiddhant/wago/pkg/queue"
	"github.com/guptasiddhant/wago/pkg/runtimecfg"
	"github.com/guptasiddhant/wago/pkg/store"
	"github.com/guptasiddhant/wago/pkg/utils"
	"github.com/guptasiddhant/wago/pkg/webhooks"
)

//go:embed all:frontend/dist
var frontendFS embed.FS

// shutdownGrace bounds how long termination waits for in-flight background
// tasks (push sends, notification emails) after the root context is cancelled.
const shutdownGrace = 5 * time.Second

func main() {
	cfg, err := utils.LoadAppConfig()
	if err != nil {
		log.Fatalf("Env Config Error: %v", err)
	}

	app := SetupApp(cfg)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

// SetupApp builds your PocketBase instance with all routes registered
func SetupApp(cfg *utils.AppConfig) *pocketbase.PocketBase {
	app := pocketbase.New()

	mgr := runtimecfg.New(cfg)
	app.OnServe().BindFunc(handleOnServe(mgr))

	return app
}

func handleOnServe(mgr *runtimecfg.Manager) func(se *core.ServeEvent) error {
	return func(se *core.ServeEvent) error {
		// Create application context with shutdown coordination.
		ac := store.NewAppContext(context.Background())

		// Store in app so handlers can access it via store.GetAppContext.
		store.SetAppContext(se.App, ac)

		// Bind to OnTerminate to cancel the root context when the server stops,
		// then give in-flight background work (pushes, notification emails)
		// a short grace period to finish before the process exits.
		se.App.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
			ac.Cancel()
			done := make(chan struct{})
			go func() {
				ac.Wait()
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(shutdownGrace):
				log.Printf("shutdown: background tasks still running after %v", shutdownGrace)
			}
			return e.Next()
		})

		// Admin credentials are always sourced from the environment (the only
		// required env var) so the superuser can always be bootstrapped.
		cfg := mgr.Env()
		if err := store.EnsureSuperuser(se.App, cfg.AdminEmail, cfg.AdminPassword); err != nil {
			log.Printf("⚠️ Superuser setup warning: %v", err)
		}
		if err := store.EnsureCollections(se.App); err != nil {
			return fmt.Errorf("Failed to setup collections: %w", err)
		}
		if err := mgr.Seed(se.App); err != nil {
			log.Printf("⚠️ Failed to seed app settings: %v", err)
		}

		// Live config so SMTP etc. reflect the persisted values each boot.
		live, err := mgr.Load(se.App)
		if err != nil {
			live = cfg
		}

		// Enable PocketBase's SMTP mailer from config so notification emails work.
		if live.SMTPHost != "" {
			settings := se.App.Settings()
			settings.SMTP = core.SMTPConfig{
				Enabled:  true,
				Host:     live.SMTPHost,
				Port:     live.SMTPPort,
				Username: live.SMTPUsername,
				Password: live.SMTPPassword,
				TLS:      live.SMTPTLS,
			}
			if err := se.App.Save(settings); err != nil {
				log.Printf("⚠️ Failed to persist SMTP settings: %v", err)
			}
		}

		notifier := notifications.NewNotifier(mgr)

		// Register Wago API routes
		api.Register(se.Router, se.App, api.Options{
			Webhook: api.WebhookConfig{
				PublicBaseURL: live.PublicBaseURL,
				VerifyToken:   live.WA_WebhookVerifyToken,
			},
			AI: aichat.Config{
				Enabled: live.AIEnabled,
				BaseURL: live.AIBaseURL,
				APIKey:  live.AIAPIKey,
				Model:   live.AIModel,
			},
			Mgr: mgr,
		})

		// Start the broadcast worker. It drains queued recipients from a
		// SQLite-backed lease queue; no cron is needed (it self-recovers).
		// The context is cancelled on server shutdown via OnTerminate hook.
		ac.Go(func(ctx context.Context) {
			queue.NewWorker(se.App, queue.Config{
				MessagesPerMinute: live.MessagesPerMinute,
				BatchSize:         live.BroadcastBatchSize,
				LeaseSeconds:      live.BroadcastLeaseSeconds,
				MaxAttempts:       live.BroadcastMaxAttempts,
			}).Run(ctx)
		})

		// Register Webhook endpoints (read config live from the manager).
		se.Router.GET("/api/wa/webhook", webhooks.HandleVerification(mgr))
		se.Router.POST("/api/wa/webhook", webhooks.HandleIncomingMessage(mgr, notifier))
		se.Router.POST("/api/wa/webhook/call", webhooks.HandleInboundCall())

		// Serve embedded React SPA from root /
		frontendSubFS, err := fs.Sub(frontendFS, "frontend/dist")
		if err != nil {
			panic(err)
		}
		se.Router.GET("/{path...}", apis.Static(frontendSubFS, true))

		return se.Next()
	}
}
