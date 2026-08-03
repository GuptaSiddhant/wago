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

	// authenticated routes (regular users + superusers)
	authed := group.Group("")
	authed.Bind(apis.RequireAuth("users", core.CollectionNameSuperusers))

	authed.GET("/auth/me", HandleMe(app))
	authed.GET("/inbox", HandleInbox(app))
	authed.GET("/contacts", HandleContacts(app))
	authed.GET("/team", HandleTeam(app))
	authed.POST("/team", HandleTeamCreate(app))
	authed.PATCH("/team/{id}", HandleTeamUpdate(app))
	authed.DELETE("/team/{id}", HandleTeamDelete(app))
	authed.GET("/accounts", HandleAccounts(app))
	authed.GET("/conversations/{id}/messages", HandleConversationMessages(app))
	authed.POST("/conversations/{id}/assign", HandleConversationAssign(app))
	authed.POST("/conversations/{id}/read", HandleConversationRead(app))
	authed.POST("/messages/send", HandleSendMessage(app))
}
