package api

import (
	"github.com/guptasiddhant/wago/pkg/aichat"
	"github.com/guptasiddhant/wago/pkg/runtimecfg"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

const (
	webhookConfigKey = "wago.api.webhook_config"
	runtimeMgrKey    = "wago.api.runtime_mgr"
	aiConfigKey      = "wago.api.ai_config"
)

// WebhookConfig carries the public-facing webhook values used when connecting
// a Meta number so Meta can deliver events to this instance.
type WebhookConfig struct {
	PublicBaseURL string // externally reachable base URL, e.g. https://wago.example.com
	VerifyToken   string // matches the token validated in HandleVerification
}

// Options bundles the optional runtime configuration Register accepts.
type Options struct {
	Webhook WebhookConfig
	AI      aichat.Config
	Mgr     *runtimecfg.Manager
}

// Register mounts all Wago API routes under /api/wa.
func Register(r *router.Router[*core.RequestEvent], app core.App, opts ...Options) {
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}
	// Store config in app.Store() so handlers can access it without globals.
	app.Store().Set(webhookConfigKey, o.Webhook)
	app.Store().Set(runtimeMgrKey, o.Mgr)
	app.Store().Set(aiConfigKey, o.AI)
	group := r.Group("/api/wa")

	// public routes
	group.POST("/auth/login", HandleLogin(app))
	group.POST("/invites/accept", HandleInviteAccept(app))
	group.GET("/invites/info", HandleInviteInfo(app))

	// authenticated routes (regular users + superusers)
	authed := group.Group("")
	authed.Bind(apis.RequireAuth("users", core.CollectionNameSuperusers))

	authed.GET("/auth/me", HandleMe(app))
	authed.PATCH("/account", HandleProfileUpdate(app))
	authed.POST("/orgs", HandleOrgCreate(app))
	authed.GET("/orgs", HandleOrgGet(app))
	authed.PATCH("/orgs", HandleOrgUpdate(app))
	authed.GET("/orgs/{id}/picture", HandleOrgPicture(app))
	authed.GET("/inbox", HandleInbox(app))
	authed.GET("/contacts", HandleContacts(app))
	authed.POST("/contacts", HandleContactCreate(app))
	authed.PATCH("/contacts/{id}", HandleContactUpdate(app))
	authed.DELETE("/contacts/{id}", HandleContactDelete(app))
	authed.GET("/team", HandleTeam(app))
	authed.POST("/team", HandleTeamCreate(app))
	authed.PATCH("/team/{id}", HandleTeamUpdate(app))
	authed.DELETE("/team/{id}", HandleTeamDelete(app))
	authed.GET("/invites", HandleInvitesList(app))
	authed.POST("/invites", HandleInviteCreate(app))
	authed.DELETE("/invites/{id}", HandleInviteRevoke(app))
	authed.GET("/teams", HandleTeamsList(app))
	authed.POST("/teams", HandleTeamsCreate(app))
	authed.PATCH("/teams/{id}", HandleTeamsUpdate(app))
	authed.DELETE("/teams/{id}", HandleTeamsDelete(app))
	authed.GET("/accounts", HandleAccounts(app))
	authed.POST("/accounts", HandleAccountCreate(app))
	authed.PATCH("/accounts/{id}", HandleProfileUpdate(app))
	authed.DELETE("/accounts/{id}", HandleAccountDelete(app))
	authed.GET("/accounts/{id}/meta", HandleAccountMeta(app))
	authed.GET("/accounts/{id}/business-profile", HandleAccountBusinessProfile(app))
	authed.POST("/accounts/{id}/business-profile/sync", HandleAccountBusinessProfileSync(app))
	authed.GET("/accounts/{id}/webhook", HandleAccountWebhookStatus(app))
	authed.POST("/accounts/{id}/webhook", HandleAccountWebhookConnect(app))
	authed.GET("/analytics", HandleAnalytics(app))
	authed.GET("/templates", HandleTemplates(app))
	authed.POST("/templates", HandleTemplateCreate(app))
	authed.POST("/templates/sync", HandleTemplatesSync(app))
	authed.POST("/templates/send", HandleTemplateSend(app))
	authed.DELETE("/templates/{id}", HandleTemplateDelete(app))
	authed.GET("/broadcasts", HandleBroadcastList(app))
	authed.POST("/broadcasts", HandleBroadcastCreate(app))
	authed.GET("/broadcasts/{id}", HandleBroadcastDetail(app))
	authed.GET("/broadcasts/{id}/events", HandleBroadcastEvents(app))
	authed.POST("/broadcasts/{id}/cancel", HandleBroadcastCancel(app))
	authed.GET("/conversations/{id}", HandleConversationDetail(app))
	authed.GET("/conversations/{id}/messages", HandleConversationMessages(app))
	authed.POST("/conversations/{id}/assign", HandleConversationAssign(app))
	authed.POST("/conversations/{id}/read", HandleConversationRead(app))
	authed.GET("/conversations/{id}/calls", HandleConversationCalls(app))
	authed.POST("/calls", HandleCallCreate(app))
	authed.POST("/calls/{id}/signal", HandleCallSignal(app))
	authed.POST("/calls/{id}/end", HandleCallEnd(app))
	authed.GET("/calls/events", HandleCallEvents(app))
	authed.POST("/messages/send", HandleSendMessage(app))
	authed.POST("/messages/media", HandleSendMediaMessage(app))
	authed.GET("/media/{wamid}", HandleMessageMedia(app))
	authed.POST("/media/upload", HandleMediaUpload(app))
	authed.POST("/ai/chat", HandleAIChat(app))
	authed.GET("/notifications", HandleNotificationsList(app))
	authed.GET("/notifications/unread-count", HandleNotificationsUnreadCount(app))
	authed.POST("/notifications/read", HandleNotificationsRead(app))
	authed.POST("/presence", HandlePresence(app))
	authed.GET("/push/config", HandlePushConfig(app))
	authed.POST("/push/subscribe", HandlePushSubscribe(app))
	authed.DELETE("/push/subscribe", HandlePushUnsubscribe(app))

	// superuser-only routes
	super := group.Group("/admin")
	super.Bind(apis.RequireSuperuserAuth())
	super.GET("/config", HandleGetConfig(app))
	super.PUT("/config", HandleUpdateConfig(app))
}

// getWebhookConfig retrieves the webhook config from app store.
func getWebhookConfig(app core.App) WebhookConfig {
	if v := app.Store().Get(webhookConfigKey); v != nil {
		if cfg, ok := v.(WebhookConfig); ok {
			return cfg
		}
	}
	return WebhookConfig{}
}

// getRuntimeMgr retrieves the runtime config manager from app store.
func getRuntimeMgr(app core.App) *runtimecfg.Manager {
	if v := app.Store().Get(runtimeMgrKey); v != nil {
		if mgr, ok := v.(*runtimecfg.Manager); ok {
			return mgr
		}
	}
	return nil
}

// getAIConfig retrieves the AI config from app store.
func getAIConfig(app core.App) aichat.Config {
	if v := app.Store().Get(aiConfigKey); v != nil {
		if cfg, ok := v.(aichat.Config); ok {
			return cfg
		}
	}
	return aichat.Config{}
}
