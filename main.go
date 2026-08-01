package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"guptasiddhant/wago/pkg/store"
	"guptasiddhant/wago/pkg/utils"
	"guptasiddhant/wago/pkg/webhooks"
)

//go:embed all:frontend/dist
var frontendFS embed.FS

func main() {
	cfg, err := utils.LoadAppConfig()
	if err != nil {
		log.Fatalf("Env Config Error: %v", err)
	}
	log.Printf("headless %v", cfg)

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

		// Register Webhook endpoints
		se.Router.GET("/api/wa/webhook", webhooks.HandleVerification(cfg.WA_WebhookVerifyToken))
		se.Router.POST("/api/wa/webhook", webhooks.HandleIncomingMessage())

		if !cfg.Headless {
			// Serve embedded React SPA from root /
			frontendSubFS, err := fs.Sub(frontendFS, "frontend/dist")
			if err != nil {
				panic(err)
			}
			se.Router.GET("/{path...}", apis.Static(frontendSubFS, true))
		}

		return se.Next()
	}
}
