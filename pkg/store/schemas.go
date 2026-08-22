package store

// RegisterCoreSchemas registers all core Wago collection schemas. The
// definitions mirror the original per-collection Ensure* functions so the
// declarative system produces identical schemas on fresh databases.
func RegisterCoreSchemas() {
	if len(schemaRegistry) > 0 {
		return // already registered (idempotent)
	}

	// orgs: business profile fields mirror the WhatsApp Business Profile API.
	RegisterSchema(NewSchemaBuilder("orgs").
		HiddenFromAPI().
		Field("name", "text", Required()).
		Field("about", "text", Max(139)).
		Field("address", "text", Max(256)).
		Field("description", "text", Max(512)).
		Field("email", "email").
		Field("websites", "json").
		Field("vertical", "select", MaxSelect(1), Values(
			"OTHER", "AUTO", "BEAUTY", "APPAREL", "EDU", "ENTERTAIN",
			"EVENT_PLAN", "FINANCE", "GROCERY", "GOVT", "HOTEL", "HEALTH",
			"NONPROFIT", "PROF_SERVICES", "RETAIL", "TRAVEL", "RESTAURANT",
		)).
		Field("profile_picture", "file", MaxSize(5<<20), MimeTypes("image/jpeg", "image/png", "image/webp")).
		AutodateFields().
		Build())

	// org_members: join table mapping users to orgs with a role. Each user can
	// list/view only their own memberships through the raw API.
	RegisterSchema(NewSchemaBuilder("org_members").
		ListRule("user = @request.auth.id").
		ViewRule("user = @request.auth.id").
		Field("org", "relation", Required(), Relation("orgs", true)).
		Field("user", "relation", Required(), Relation("users", true)).
		Field("role", "select", Required(), MaxSelect(1), Values(AllRoles...)).
		AutodateFields().
		Index("idx_org_members_unique", true, "org, user", "").
		Build())

	// teams: org-scoped agent groups for round-robin routing.
	RegisterSchema(NewSchemaBuilder("teams").
		HiddenFromAPI().
		Field("org", "relation", Required(), Relation("orgs", true)).
		Field("name", "text", Required()).
		AutodateFields().
		Index("idx_teams_org_name", true, "org, name", "").
		Build())

	// whatsapp_accounts: Meta credentials for each connected number.
	RegisterSchema(NewSchemaBuilder("whatsapp_accounts").
		HiddenFromAPI().
		Field("org", "relation", Required(), Relation("orgs", true)).
		Field("display_name", "text").
		Field("waba_id", "text").
		Field("phone_number_id", "text", Required()).
		Field("access_token", "text", Required()).
		Field("verify_token", "text").
		Field("status", "select", MaxSelect(1), Values("connected", "disconnected")).
		AutodateFields().
		Index("idx_whatsapp_accounts_phone_number_id", true, "phone_number_id", "").
		Build())

	// contacts: the people each org messages plus their tags/notes.
	RegisterSchema(NewSchemaBuilder("contacts").
		HiddenFromAPI().
		Field("org", "relation", Required(), Relation("orgs", true)).
		Field("phone", "text", Required()).
		Field("name", "text").
		Field("tags", "json").
		Field("notes", "text").
		Field("last_activity", "date").
		AutodateFields().
		Index("idx_contacts_org_phone", true, "org, phone", "").
		Build())

	// conversations: one thread per (org, contact, account) with assignment state.
	RegisterSchema(NewSchemaBuilder("conversations").
		HiddenFromAPI().
		Field("org", "relation", Required(), Relation("orgs", true)).
		Field("contact", "relation", Required(), Relation("contacts", true)).
		Field("whatsapp_account", "relation", Required(), Relation("whatsapp_accounts", true)).
		Field("assignee", "relation", Relation("users", false)).
		Field("unread_count", "number").
		Field("last_message_at", "date").
		Field("status", "select", MaxSelect(1), Values("open", "closed")).
		AutodateFields().
		Index("idx_conversations_unique", true, "org, contact, whatsapp_account", "").
		Build())

	// messages: every inbound/outbound WhatsApp message plus delivery status.
	RegisterSchema(NewSchemaBuilder("messages").
		HiddenFromAPI().
		Field("org", "relation", Required(), Relation("orgs", true)).
		Field("conversation", "relation", Required(), Relation("conversations", true)).
		Field("wamid", "text", Required()).
		Field("sender_phone", "text", Required()).
		Field("recipient_phone", "text", Required()).
		Field("body", "text").
		Field("direction", "select", Required(), MaxSelect(1), Values("inbound", "outbound")).
		Field("status", "select", MaxSelect(1), Values("sent", "delivered", "read", "failed")).
		Field("payload", "json", Required()).
		Field("media", "file", MaxSize(100<<20)).
		AutodateFields().
		Index("idx_messages_wamid", true, "wamid", "").
		Build())

	// invites: pending/expired team-member onboarding invitations.
	RegisterSchema(NewSchemaBuilder("invites").
		HiddenFromAPI().
		Field("org", "relation", Required(), Relation("orgs", true)).
		Field("team", "relation", Relation("teams", false)).
		Field("created_by", "relation", Relation("users", false)).
		Field("email", "text", Required()).
		Field("role", "select", Required(), MaxSelect(1), Values(AllRoles...)).
		Field("token", "text", Required()).
		Field("status", "select", Required(), MaxSelect(1),
			Values(InviteStatusPending, InviteStatusAccepted, InviteStatusRevoked)).
		Field("expires_at", "date").
		AutodateFields().
		Index("idx_invites_token", true, "token", "").
		Index("idx_invites_org_email", false, "org, email", "").
		Build())

	// notifications: unread push/email/WhatsApp alerts for users.
	RegisterSchema(NewSchemaBuilder("notifications").
		HiddenFromAPI().
		Field("org", "relation", Required(), Relation("orgs", true)).
		Field("user", "relation", Required(), Relation("users", true)).
		Field("conversation", "relation", Required(), Relation("conversations", true)).
		Field("kind", "text").
		Field("body", "text").
		Field("read", "bool").
		AutodateFields().
		Index("idx_notifications_user_created", false, "user, created", "").
		Build())

	// push_subscriptions: Web Push device subscriptions per user.
	RegisterSchema(NewSchemaBuilder(PushSubscriptionsC).
		HiddenFromAPI().
		Field("org", "relation", Required(), Relation("orgs", true)).
		Field("user", "relation", Required(), Relation("users", true)).
		Field("endpoint", "text", Required()).
		Field("auth", "text").
		Field("p256dh", "text").
		Field("created", "autodate", OnCreate(true)).
		Index("idx_push_sub_user_endpoint", false, "user, endpoint", "").
		Build())

	// message_templates: templates cached locally and synced with Meta.
	RegisterSchema(NewSchemaBuilder("message_templates").
		HiddenFromAPI().
		Field("org", "relation", Required(), Relation("orgs", true)).
		Field("account", "relation", Required(), Relation("whatsapp_accounts", true)).
		Field("meta_id", "text").
		Field("name", "text", Required()).
		Field("language", "text", Required()).
		Field("category", "select", MaxSelect(1), Values("MARKETING", "UTILITY", "AUTHENTICATION")).
		Field("header_type", "text").
		Field("header_text", "text").
		Field("header_media_type", "text").
		Field("header_media_id", "text").
		Field("header_media_name", "text").
		Field("body", "text", Required()).
		Field("footer", "text").
		Field("buttons", "json").
		Field("status", "text", Required()).
		Field("meta_error", "text").
		AutodateFields().
		Index("idx_msg_templates_meta_id", true, "meta_id", "").
		Build())

	// broadcasts: broadcast header record driven by the queue worker.
	RegisterSchema(NewSchemaBuilder("broadcasts").
		HiddenFromAPI().
		Field("org", "relation", Required(), Relation("orgs", true)).
		Field("account", "relation", Required(), Relation("whatsapp_accounts", true)).
		Field("template", "relation", Required(), Relation("message_templates", true)).
		Field("created_by", "relation", Relation("users", false)).
		Field("name", "text", Required()).
		Field("status", "select", MaxSelect(1),
			Values(BroadcastQueued, BroadcastRunning, BroadcastCompleted, BroadcastFailed, BroadcastCancelled)).
		Field("params", "json").
		Field("header_media_type", "text").
		Field("header_media_id", "text").
		Field("header_media_name", "text").
		Field("rate_per_minute", "number").
		Field("batch_size", "number").
		Field("recipient_count", "number").
		Field("sent_count", "number").
		Field("failed_count", "number").
		Field("started_at", "date").
		Field("finished_at", "date").
		AutodateFields().
		Build())

	// broadcast_recipients: the per-recipient lease queue drained by the worker.
	RegisterSchema(NewSchemaBuilder("broadcast_recipients").
		HiddenFromAPI().
		Field("org", "relation", Required(), Relation("orgs", true)).
		Field("broadcast", "relation", Required(), Relation("broadcasts", true)).
		Field("contact", "relation", Required(), Relation("contacts", true)).
		Field("phone", "text", Required()).
		Field("name", "text").
		Field("status", "select", MaxSelect(1),
			Values(RecipientQueued, RecipientSending, RecipientSent, RecipientFailed)).
		Field("attempts", "number").
		Field("next_attempt_at", "date").
		Field("lease_until", "date").
		Field("wamid", "text").
		Field("error", "text").
		Field("sent_at", "date").
		AutodateFields().
		Index("idx_bc_recipients_broadcast", false, "broadcast", "").
		Index("idx_bc_recipients_status", false, "broadcast, status", "").
		Build())

	// voice_calls: every in/outbound call against a conversation.
	RegisterSchema(NewSchemaBuilder("voice_calls").
		HiddenFromAPI().
		Field("org", "relation", Required(), Relation("orgs", true)).
		Field("account", "relation", Required(), Relation("whatsapp_accounts", true)).
		Field("contact", "relation", Required(), Relation("contacts", true)).
		Field("conversation", "relation", Required(), Relation("conversations", true)).
		Field("direction", "text", Required()).
		Field("status", "text", Required()).
		Field("phone", "text", Required()).
		Field("peer_name", "text").
		Field("started_at", "date").
		Field("answered_at", "date").
		Field("ended_at", "date").
		Field("duration", "number").
		AutodateFields().
		Index("idx_voice_calls_conversation", false, "conversation", "").
		Build())

	// app_settings: singleton runtime-editable configuration + VAPID keys.
	RegisterSchema(NewSchemaBuilder(settingsCollectionName).
		HiddenFromAPI().
		Field("config", "json").
		Field("vapid_public_key", "text").
		Field("vapid_private_key", "text").
		AutodateFields().
		Build())
}