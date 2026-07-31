package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"guptasiddhant/wago/pkg/store"
	"guptasiddhant/wago/pkg/webhooks"
)

//go:embed all:frontend/dist
var frontendFS embed.FS

func main() {

	if err := godotenv.Load(); err != nil {
		// Log a warning or error if .env is missing
		log.Println("Warning: No .env file found, reading from environment")
	}

	app := pocketbase.New()

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		email := os.Getenv("ADMIN_EMAIL")
		password := os.Getenv("ADMIN_PASSWORD")

		if err := store.EnsureSuperuser(se.App, email, password); err != nil {
			log.Printf("⚠️ Superuser setup warning: %v", err)
		}
		if err := store.EnsureCollections(se.App); err != nil {
			return fmt.Errorf("Failed to setup collections: %w", err)
		}

		// Register Webhook endpoints
		se.Router.GET("/api/wa/webhook", webhooks.HandleVerification())
		se.Router.POST("/api/wa/webhook", webhooks.HandleIncomingMessage(se.App))

		// Serve embedded React SPA from root /
		se.Router.GET("/{path...}", apis.Static(getReactFS(), true))

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

// Helper to extract the sub-filesystem without the leading "dist" folder prefix
func getReactFS() fs.FS {
	reactSubFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		panic(err)
	}
	return reactSubFS
}
