package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/guptasiddhant/wago/pkg/api"
	"github.com/guptasiddhant/wago/pkg/notifications"
	"github.com/guptasiddhant/wago/pkg/store"
	"github.com/guptasiddhant/wago/pkg/utils"
	"github.com/guptasiddhant/wago/pkg/webhooks"
)

//go:embed all:frontend/dist
var frontendFS embed.FS

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

	app.OnServe().BindFunc(handleOnServe(cfg))

	return app
}

func handleOnServe(cfg *utils.AppConfig) func(se *core.ServeEvent) error {
	return func(se *core.ServeEvent) error {
		if err := store.EnsureSuperuser(se.App, cfg.AdminEmail, cfg.AdminPassword); err != nil {
			log.Printf("⚠️ Superuser setup warning: %v", err)
		}
		if err := store.EnsureCollections(se.App); err != nil {
			return fmt.Errorf("Failed to setup collections: %w", err)
		}

		// Enable PocketBase's SMTP mailer from config so notification emails work.
		if cfg.SMTPHost != "" {
			settings := se.App.Settings()
			settings.SMTP = core.SMTPConfig{
				Enabled:  true,
				Host:     cfg.SMTPHost,
				Port:     cfg.SMTPPort,
				Username: cfg.SMTPUsername,
				Password: cfg.SMTPPassword,
				TLS:      cfg.SMTPTLS,
			}
			if err := se.App.Save(settings); err != nil {
				log.Printf("⚠️ Failed to persist SMTP settings: %v", err)
			}
		}

		notifier := notifications.NewNotifier(cfg)

		// Register Wago API routes
		api.Register(se.Router, se.App)

		// Register Webhook endpoints
		se.Router.GET("/api/wa/webhook", webhooks.HandleVerification(cfg.WA_WebhookVerifyToken))
		se.Router.POST("/api/wa/webhook", webhooks.HandleIncomingMessage(cfg.MetaAppSecret, notifier))

		// Serve embedded React SPA from root /
		frontendSubFS, err := fs.Sub(frontendFS, "frontend/dist")
		if err != nil {
			panic(err)
		}
		se.Router.GET("/{path...}", apis.Static(frontendSubFS, true))

		return se.Next()
	}
}
