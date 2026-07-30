package main

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"guptasiddhant/wago/pkg/store"
	"guptasiddhant/wago/pkg/webhooks"
)

func main() {
	app := pocketbase.New()

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if err := store.EnsureSuperuser(se.App); err != nil {
			log.Printf("⚠️ Superuser setup warning: %v", err)
		}
		if err := store.EnsureCollections(se.App); err != nil {
			return fmt.Errorf("Failed to setup collections: %w", err)
		}

		// Register Webhook endpoints
		se.Router.GET("/api/wa/webhook", webhooks.HandleVerification())
		se.Router.POST("/api/wa/webhook", webhooks.HandleIncomingMessage(se.App))

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
