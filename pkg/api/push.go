package api

import (
	"net/http"

	"github.com/SherClockHolmes/webpush-go"

	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
)

// HandlePushConfig exposes the VAPID public key needed to subscribe the
// browser's push service.
func HandlePushConfig(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		publicKey, _, err := store.GetVAPIDKeys(app, webpush.GenerateVAPIDKeys)
		if err != nil {
			return e.InternalServerError("Failed to prepare push keys", err)
		}
		return e.JSON(http.StatusOK, map[string]any{"vapid_public_key": publicKey})
	}
}

type pushSubscribeRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		Auth   string `json:"auth"`
		P256dh string `json:"p256dh"`
	} `json:"keys"`
}

// HandlePushSubscribe stores a Web Push device subscription for the current
// user (replacing any existing subscription with the same endpoint).
func HandlePushSubscribe(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		var body pushSubscribeRequest
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("Invalid push subscription", err)
		}
		if body.Endpoint == "" || body.Keys.P256dh == "" || body.Keys.Auth == "" {
			return e.BadRequestError("Push subscription requires endpoint, p256dh and auth", nil)
		}

		if err := store.SavePushSubscription(app, access.OrgID, e.Auth.Id, body.Endpoint, body.Keys.Auth, body.Keys.P256dh); err != nil {
			return e.InternalServerError("Failed to store push subscription", err)
		}
		return e.NoContent(http.StatusNoContent)
	}
}

// HandlePushUnsubscribe removes a Web Push subscription for the current user.
func HandlePushUnsubscribe(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		endpoint := e.Request.URL.Query().Get("endpoint")
		if endpoint == "" {
			return e.BadRequestError("endpoint query parameter is required", nil)
		}

		if err := store.DeletePushSubscription(app, access.OrgID, e.Auth.Id, endpoint); err != nil {
			return e.InternalServerError("Failed to remove push subscription", err)
		}
		return e.NoContent(http.StatusNoContent)
	}
}
