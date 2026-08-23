package store

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// UpsertConversation finds or creates the conversation for org+contact+account
// and bumps its last_message_at. It reports whether a new conversation was
// created (used to trigger round-robin assignment).
func UpsertConversation(app core.App, orgID, contactID, accountID string, ts time.Time) (*core.Record, bool, error) {
	conv, err := app.FindFirstRecordByFilter("conversations",
		"org = {:org} && contact = {:contact} && whatsapp_account = {:account}",
		DbxParams(map[string]any{"org": orgID, "contact": contactID, "account": accountID}))
	if err == nil {
		conv.Set("last_message_at", ts)
		if err := app.Save(conv); err != nil {
			return nil, false, fmt.Errorf("failed to update conversation: %w", err)
		}
		return conv, false, nil
	}

	// Route new conversations to the whatsapp account's team, if any.
	teamID := ""
	if account, err := app.FindRecordById("whatsapp_accounts", accountID); err == nil {
		teamID = account.GetString("team")
	}

	convCol, err := app.FindCollectionByNameOrId("conversations")
	if err != nil {
		return nil, false, err
	}
	conv = core.NewRecord(convCol)
	conv.Set("org", orgID)
	conv.Set("contact", contactID)
	conv.Set("whatsapp_account", accountID)
	conv.Set("team", teamID)
	conv.Set("status", "open")
	conv.Set("unread_count", 0)
	conv.Set("last_message_at", ts)
	if err := app.Save(conv); err != nil {
		return nil, false, fmt.Errorf("failed to upsert conversation: %w", err)
	}
	return conv, true, nil
}

// IncrementConversationUnread increments the unread counter of a conversation.
func IncrementConversationUnread(app core.App, convID string) error {
	conv, err := app.FindRecordById("conversations", convID)
	if err != nil {
		return err
	}
	conv.Set("unread_count", conv.GetInt("unread_count")+1)
	return app.Save(conv)
}

// MarkConversationRead resets the unread counter to zero.
func MarkConversationRead(app core.App, orgID, convID string) error {
	conv, err := FindOrgRecord(app, orgID, "conversations", convID)
	if err != nil {
		return err
	}
	conv.Set("unread_count", 0)
	return app.Save(conv)
}

// CountAssigneeLoad returns how many conversations a user is currently assigned
// to. When teamID is non-empty the count is scoped to that team.
func CountAssigneeLoad(app core.App, orgID, teamID, userID string) (int, error) {
	filter := "org = {:org} && assignee = {:user}"
	params := map[string]any{"org": orgID, "user": userID}
	if teamID != "" {
		filter += " && team = {:team}"
		params["team"] = teamID
	}
	records, err := app.FindRecordsByFilter("conversations", filter, "", 1000, 0, DbxParams(params))
	if err != nil {
		return 0, err
	}
	return len(records), nil
}

// AssignConversationRR assigns a conversation to the least-loaded eligible
// agent (round-robin) and returns the chosen assignee id.
func AssignConversationRR(app core.App, conv *core.Record) (string, error) {
	assignee, err := PickRoundRobinAssignee(app, conv.GetString("org"), conv.GetString("team"))
	if err != nil {
		return "", err
	}
	conv.Set("assignee", assignee)
	if err := app.Save(conv); err != nil {
		return "", err
	}
	return assignee, nil
}

// PickRoundRobinAssignee returns the eligible org member with the fewest
// currently assigned conversations, distributing work evenly across the team.
// Eligible roles are owner, admin, and agent (viewers are excluded). When the
// conversation is routed to a team, only that team's members are considered;
// untagged conversations consider every eligible member of the org.
func PickRoundRobinAssignee(app core.App, orgID, teamID string) (string, error) {
	filter := "org = {:org} && (role = 'owner' || role = 'admin' || role = 'agent')"
	params := map[string]any{"org": orgID}
	if teamID != "" {
		filter += " && team = {:team}"
		params["team"] = teamID
	}

	members, err := app.FindRecordsByFilter("org_members", filter, "created", 500, 0, DbxParams(params))
	if err != nil {
		return "", err
	}
	if len(members) == 0 {
		return "", fmt.Errorf("no eligible agents to assign to")
	}

	best := ""
	bestLoad := int(^uint(0) >> 1) // MaxInt
	for _, m := range members {
		userID := m.GetString("user")
		load, err := CountAssigneeLoad(app, orgID, teamID, userID)
		if err != nil {
			continue
		}
		if load < bestLoad {
			bestLoad = load
			best = userID
		}
	}
	if best == "" {
		best = members[0].GetString("user")
	}
	return best, nil
}
