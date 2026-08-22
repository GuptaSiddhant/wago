package api

import (
	"net/http"

	"github.com/guptasiddhant/wago/pkg/utils"

	"github.com/pocketbase/pocketbase/core"
)

// configRequest is the editable subset of the Wago configuration exposed to a
// superadmin through the API. Sensitive admin credentials are intentionally
// excluded (they are env-only).
type configRequest struct {
	WA_WebhookVerifyToken string `json:"wa_webhook_verify_token"`
	MetaAppSecret         string `json:"meta_app_secret"`
	PublicBaseURL         string `json:"public_base_url"`

	AIEnabled bool   `json:"ai_enabled"`
	AIBaseURL string `json:"ai_base_url"`
	AIAPIKey  string `json:"ai_api_key"`
	AIModel   string `json:"ai_model"`

	SMTPHost        string `json:"smtp_host"`
	SMTPPort        int    `json:"smtp_port"`
	SMTPUsername    string `json:"smtp_username"`
	SMTPPassword    string `json:"smtp_password"`
	SMTPTLS         bool   `json:"smtp_tls"`
	SMTPFromAddress string `json:"smtp_from_address"`
	SMTPFromName    string `json:"smtp_from_name"`

	VAPIDSubject            string `json:"vapid_subject"`
	WA_NotificationTemplate string `json:"wa_notification_template"`

	MessagesPerMinute     int `json:"messages_per_minute"`
	BroadcastBatchSize    int `json:"broadcast_batch_size"`
	BroadcastLeaseSeconds int `json:"broadcast_lease_seconds"`
	BroadcastMaxAttempts  int `json:"broadcast_max_attempts"`
}

// HandleGetConfig returns the current runtime configuration to a superadmin.
func HandleGetConfig(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		mgr := getRuntimeMgr(app)
		if mgr == nil {
			return e.InternalServerError("Configuration manager not available", nil)
		}
		cfg, err := mgr.Load(app)
		if err != nil {
			return e.InternalServerError("Failed to load configuration", err)
		}
		return e.JSON(http.StatusOK, toConfigRequest(cfg))
	}
}

// HandleUpdateConfig persists the editable runtime configuration. It is
// superuser-only and applies immediately.
func HandleUpdateConfig(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var body configRequest
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("Invalid request body", err)
		}

		mgr := getRuntimeMgr(app)
		if mgr == nil {
			return e.InternalServerError("Configuration manager not available", nil)
		}
		cfg, err := mgr.Load(app)
		if err != nil {
			return e.InternalServerError("Failed to load configuration", err)
		}

		// Map the submitted fields onto the full config, keeping env-only admin
		// credentials untouched.
		cfg.WA_WebhookVerifyToken = body.WA_WebhookVerifyToken
		cfg.MetaAppSecret = body.MetaAppSecret
		cfg.PublicBaseURL = body.PublicBaseURL
		cfg.AIEnabled = body.AIEnabled
		cfg.AIBaseURL = body.AIBaseURL
		cfg.AIAPIKey = body.AIAPIKey
		cfg.AIModel = body.AIModel
		cfg.SMTPHost = body.SMTPHost
		cfg.SMTPPort = body.SMTPPort
		cfg.SMTPUsername = body.SMTPUsername
		cfg.SMTPPassword = body.SMTPPassword
		cfg.SMTPTLS = body.SMTPTLS
		cfg.SMTPFromAddress = body.SMTPFromAddress
		cfg.SMTPFromName = body.SMTPFromName
		cfg.VAPIDSubject = body.VAPIDSubject
		cfg.WA_NotificationTemplate = body.WA_NotificationTemplate
		cfg.MessagesPerMinute = body.MessagesPerMinute
		cfg.BroadcastBatchSize = body.BroadcastBatchSize
		cfg.BroadcastLeaseSeconds = body.BroadcastLeaseSeconds
		cfg.BroadcastMaxAttempts = body.BroadcastMaxAttempts

		if err := mgr.Save(app, cfg); err != nil {
			return e.InternalServerError("Failed to save configuration", err)
		}

		return e.JSON(http.StatusOK, toConfigRequest(cfg))
	}
}

func toConfigRequest(cfg *utils.AppConfig) configRequest {
	return configRequest{
		WA_WebhookVerifyToken: cfg.WA_WebhookVerifyToken,
		MetaAppSecret:         cfg.MetaAppSecret,
		PublicBaseURL:         cfg.PublicBaseURL,
		AIEnabled:             cfg.AIEnabled,
		AIBaseURL:             cfg.AIBaseURL,
		AIAPIKey:              cfg.AIAPIKey,
		AIModel:               cfg.AIModel,
		SMTPHost:              cfg.SMTPHost,
		SMTPPort:              cfg.SMTPPort,
		SMTPUsername:          cfg.SMTPUsername,
		SMTPPassword:          cfg.SMTPPassword,
		SMTPTLS:               cfg.SMTPTLS,
		SMTPFromAddress:       cfg.SMTPFromAddress,
		SMTPFromName:          cfg.SMTPFromName,
		VAPIDSubject:          cfg.VAPIDSubject,
		WA_NotificationTemplate: cfg.WA_NotificationTemplate,
		MessagesPerMinute:     cfg.MessagesPerMinute,
		BroadcastBatchSize:    cfg.BroadcastBatchSize,
		BroadcastLeaseSeconds: cfg.BroadcastLeaseSeconds,
		BroadcastMaxAttempts:  cfg.BroadcastMaxAttempts,
	}
}
