package store

import (
	"log"

	"github.com/SherClockHolmes/webpush-go"

	"github.com/pocketbase/pocketbase/core"
)

const (
	// PushSubscriptionsC stores Web Push device subscriptions per user.
	PushSubscriptionsC = "push_subscriptions"
)

// EnsurePushCollections creates the push_subscriptions collection on first boot.
func EnsurePushCollections(app core.App) error {
	if _, err := app.FindCollectionByNameOrId(PushSubscriptionsC); err != nil {
		orgsCol, err := app.FindCollectionByNameOrId("orgs")
		if err != nil {
			return err
		}
		usersCol, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		col := core.NewBaseCollection(PushSubscriptionsC)
		// Device subscriptions are only reachable through the scoped Wago API
		// (and PB superusers which bypass rules).
		col.ListRule = nil
		col.ViewRule = nil
		col.CreateRule = nil
		col.UpdateRule = nil
		col.DeleteRule = nil

		col.Fields.Add(
			&core.RelationField{
				Name:          "org",
				CollectionId:  orgsCol.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.RelationField{
				Name:          "user",
				CollectionId:  usersCol.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.TextField{Name: "endpoint", Required: true},
			&core.TextField{Name: "auth"},
			&core.TextField{Name: "p256dh"},
			&core.AutodateField{Name: "created", OnCreate: true})

		col.AddIndex("idx_push_sub_user_endpoint", false, "user, endpoint", "")
		if err := app.Save(col); err != nil {
			return err
		}
		log.Println("Auto-created 'push_subscriptions' collection")
	}

	return nil
}

// MigrateLegacyVAPIDKeys moves an existing keypair from the old vapid_keys
// collection into the app_settings record and drops the legacy collection. It
// is a no-op when the legacy collection was never created.
func MigrateLegacyVAPIDKeys(app core.App) error {
	col, err := app.FindCollectionByNameOrId("vapid_keys")
	if err != nil {
		return nil // never created
	}

	rec, findErr := app.FindFirstRecordByFilter("vapid_keys", "", nil)
	if findErr == nil && rec.GetString("public_key") != "" {
		if err := SaveVAPIDKeysToSettings(app, rec.GetString("public_key"), rec.GetString("private_key")); err != nil {
			return err
		}
	}

	if err := app.Delete(col); err != nil {
		return err
	}
	log.Println("Migrated 'vapid_keys' into 'app_settings' and removed the legacy collection")
	return nil
}

// SavePushSubscription stores (or replaces) a device subscription for a user.
func SavePushSubscription(app core.App, orgID, userID, endpoint, auth, p256dh string) error {
	existing, err := app.FindFirstRecordByFilter(PushSubscriptionsC,
		"user = {:user} && endpoint = {:endpoint}",
		DbxParams(map[string]any{"user": userID, "endpoint": endpoint}))
	if err == nil {
		if err := app.Delete(existing); err != nil {
			return err
		}
	}

	col, err := app.FindCollectionByNameOrId(PushSubscriptionsC)
	if err != nil {
		return err
	}
	rec := core.NewRecord(col)
	rec.Set("org", orgID)
	rec.Set("user", userID)
	rec.Set("endpoint", endpoint)
	rec.Set("auth", auth)
	rec.Set("p256dh", p256dh)
	return app.Save(rec)
}

// DeletePushSubscription removes a specific device subscription for a user.
func DeletePushSubscription(app core.App, orgID, userID, endpoint string) error {
	rec, err := app.FindFirstRecordByFilter(PushSubscriptionsC,
		"org = {:org} && user = {:user} && endpoint = {:endpoint}",
		DbxParams(map[string]any{"org": orgID, "user": userID, "endpoint": endpoint}))
	if err != nil {
		return nil
	}
	return app.Delete(rec)
}

// DeletePushSubscriptionByID removes a subscription by record id (used when a
// push service reports the subscription as expired).
func DeletePushSubscriptionByID(app core.App, orgID, id string) error {
	rec, err := FindOrgRecord(app, orgID, PushSubscriptionsC, id)
	if err != nil {
		return nil
	}
	return app.Delete(rec)
}

// ListPushSubscriptions returns all device subscriptions for a user.
func ListPushSubscriptions(app core.App, orgID, userID string) ([]*core.Record, error) {
	return app.FindRecordsByFilter(PushSubscriptionsC,
		"org = {:org} && user = {:user}",
		"created", 100, 0,
		DbxParams(map[string]any{"org": orgID, "user": userID}))
}

// GetVAPIDKeys returns the app-wide VAPID keypair, generating and storing them
// on first request if they don't exist yet. Keys live on the singleton
// app_settings record.
func GetVAPIDKeys(app core.App) (publicKey, privateKey string, err error) {
	publicKey, privateKey, err = GetVAPIDKeysFromSettings(app)
	if err != nil {
		return "", "", err
	}
	if publicKey != "" {
		return publicKey, privateKey, nil
	}

	privateKey, publicKey, genErr := webpush.GenerateVAPIDKeys()
	if genErr != nil {
		return "", "", genErr
	}
	if err := SaveVAPIDKeysToSettings(app, publicKey, privateKey); err != nil {
		return "", "", err
	}
	return publicKey, privateKey, nil
}
