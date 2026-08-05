package api

import (
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

// Register mounts all Wago API routes under /api/wa.
func Register(r *router.Router[*core.RequestEvent], app core.App) {
	group := r.Group("/api/wa")

	// public routes
	group.POST("/auth/login", HandleLogin(app))
	group.POST("/invites/accept", HandleInviteAccept(app))
	group.GET("/invites/info", HandleInviteInfo(app))

	// authenticated routes (regular users + superusers)
	authed := group.Group("")
	authed.Bind(apis.RequireAuth("users", core.CollectionNameSuperusers))

	authed.GET("/auth/me", HandleMe(app))
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
	authed.PATCH("/accounts/{id}", HandleAccountUpdate(app))
	authed.DELETE("/accounts/{id}", HandleAccountDelete(app))
	authed.GET("/accounts/{id}/meta", HandleAccountMeta(app))
	authed.GET("/analytics", HandleAnalytics(app))
	authed.GET("/templates", HandleTemplates(app))
	authed.POST("/templates", HandleTemplateCreate(app))
	authed.POST("/templates/sync", HandleTemplatesSync(app))
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
	authed.POST("/messages/send", HandleSendMessage(app))
	authed.GET("/notifications", HandleNotificationsList(app))
	authed.GET("/notifications/unread-count", HandleNotificationsUnreadCount(app))
	authed.POST("/notifications/read", HandleNotificationsRead(app))
	authed.POST("/presence", HandlePresence(app))
	authed.GET("/push/config", HandlePushConfig(app))
	authed.POST("/push/subscribe", HandlePushSubscribe(app))
	authed.DELETE("/push/subscribe", HandlePushUnsubscribe(app))
}
