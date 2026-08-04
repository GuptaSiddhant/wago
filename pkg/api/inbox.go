package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

type contactDTO struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Phone string   `json:"phone"`
	Tags  []string `json:"tags"`
	Notes string   `json:"notes"`
}

// contactToDTO maps a contacts record into its wire representation.
func contactToDTO(contact *core.Record) contactDTO {
	return contactDTO{
		ID:    contact.Id,
		Name:  contact.GetString("name"),
		Phone: contact.GetString("phone"),
		Tags:  contact.GetStringSlice("tags"),
		Notes: contact.GetString("notes"),
	}
}

type accountDTO struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type messageDTO struct {
	ID        string `json:"id"`
	Wamid     string `json:"wamid"`
	Body      string `json:"body"`
	Direction string `json:"direction"`
	Status    string `json:"status"`
	Created   string `json:"created"`
}

type conversationDTO struct {
	ID              string      `json:"id"`
	Contact         contactDTO  `json:"contact"`
	WhatsAppAccount accountDTO  `json:"whatsapp_account"`
	AssigneeID      string      `json:"assignee_id"`
	UnreadCount     int         `json:"unread_count"`
	LastMessageAt   string      `json:"last_message_at"`
	Status          string      `json:"status"`
	LastMessage     *messageDTO `json:"last_message,omitempty"`
}

func conversationToDTO(app core.App, conv *core.Record) (*conversationDTO, error) {
	dto := &conversationDTO{
		ID:          conv.Id,
		AssigneeID:  conv.GetString("assignee"),
		UnreadCount: conv.GetInt("unread_count"),
		Status:      conv.GetString("status"),
	}

	dto.LastMessageAt = conv.GetDateTime("last_message_at").Time().UTC().Format(types.DefaultDateLayout)

	contact, err := app.FindRecordById("contacts", conv.GetString("contact"))
	if err == nil {
		dto.Contact = contactToDTO(contact)
	}

	acc, err := app.FindRecordById("whatsapp_accounts", conv.GetString("whatsapp_account"))
	if err == nil {
		dto.WhatsAppAccount = accountDTO{
			ID:          acc.Id,
			DisplayName: acc.GetString("display_name"),
		}
	}

	last, err := app.FindRecordsByFilter("messages",
		"conversation = {:conv}", "-created", 1, 0,
		store.DbxParams(map[string]any{"conv": conv.Id}))
	if err == nil && len(last) > 0 {
		dto.LastMessage = messageToDTO(last[0])
	}

	return dto, nil
}

func messageToDTO(msg *core.Record) *messageDTO {
	dto := &messageDTO{
		ID:        msg.Id,
		Wamid:     msg.GetString("wamid"),
		Body:      msg.GetString("body"),
		Direction: msg.GetString("direction"),
		Status:    msg.GetString("status"),
	}
	dto.Created = msg.GetDateTime("created").Time().UTC().Format(types.DefaultDateLayout)
	return dto
}

// HandleInbox lists conversations for the current org.
func HandleInbox(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		q := e.Request.URL.Query()
		search := strings.TrimSpace(q.Get("search"))
		assignee := strings.TrimSpace(q.Get("assignee"))
		status := strings.TrimSpace(q.Get("status"))

		limit, _ := strconv.Atoi(q.Get("limit"))
		if limit <= 0 || limit > 200 {
			limit = 50
		}
		offset, _ := strconv.Atoi(q.Get("offset"))
		if offset < 0 {
			offset = 0
		}

		params := map[string]any{"org": access.OrgID}
		filter := "org = {:org}"
		if !access.CanSeeAllConversations() {
			// Agents/viewers only see conversations routed to their team plus
			// conversations that aren't routed to any team yet.
			params["team"] = access.TeamID
			filter += " && (team = {:team} || team = null)"
		}
		if search != "" {
			params["q"] = search
			filter += " && (contact.name ~ {:q} || contact.phone ~ {:q})"
		}
		if assignee != "" {
			params["assignee"] = assignee
			filter += " && assignee = {:assignee}"
		} else if strings.EqualFold(q.Get("unassigned"), "true") {
			filter += " && assignee = ''"
		}
		if status != "" {
			params["status"] = status
			filter += " && status = {:status}"
		}

		records, err := app.FindRecordsByFilter("conversations", filter, "-last_message_at", limit, offset,
			store.DbxParams(params))
		if err != nil {
			return e.InternalServerError("Failed to list conversations", err)
		}

		items := make([]*conversationDTO, 0, len(records))
		for _, r := range records {
			dto, err := conversationToDTO(app, r)
			if err != nil {
				continue
			}
			items = append(items, dto)
		}

		return e.JSON(http.StatusOK, map[string]any{"items": items})
	}
}

// HandleConversationMessages lists messages for a conversation.
func HandleConversationMessages(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		convID := e.Request.PathValue("id")
		conv, err := app.FindRecordById("conversations", convID)
		if err != nil {
			return e.NotFoundError("conversation not found", nil)
		}
		if conv.GetString("org") != access.OrgID || !access.CanViewTeam(conv.GetString("team")) {
			return e.ForbiddenError("you don't have access to this conversation", nil)
		}

		q := e.Request.URL.Query()
		limit, _ := strconv.Atoi(q.Get("limit"))
		if limit <= 0 || limit > 200 {
			limit = 50
		}
		offset, _ := strconv.Atoi(q.Get("offset"))
		if offset < 0 {
			offset = 0
		}

		records, err := app.FindRecordsByFilter("messages",
			"conversation = {:conv}", "-created", limit, offset,
			store.DbxParams(map[string]any{"conv": convID}))
		if err != nil {
			return e.InternalServerError("Failed to list messages", err)
		}

		items := make([]*messageDTO, 0, len(records))
		for _, r := range records {
			items = append(items, messageToDTO(r))
		}

		return e.JSON(http.StatusOK, map[string]any{"items": items})
	}
}

// conversationDetailDTO extends the list representation with the full contact,
// assignee/team context, and the Meta service-window state.
type conversationDetailDTO struct {
	conversationDTO
	AssigneeName string `json:"assignee_name,omitempty"`
	TeamID       string `json:"team_id,omitempty"`
	TeamName     string `json:"team_name,omitempty"`
	InWindow     bool   `json:"in_window"`
}

// HandleConversationDetail returns full detail for a single conversation,
// including the contact (tags/notes) and the 24h service-window state.
func HandleConversationDetail(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		convID := e.Request.PathValue("id")
		conv, err := app.FindRecordById("conversations", convID)
		if err != nil {
			return e.NotFoundError("conversation not found", nil)
		}
		if conv.GetString("org") != access.OrgID || !access.CanViewTeam(conv.GetString("team")) {
			return e.ForbiddenError("you don't have access to this conversation", nil)
		}

		base, err := conversationToDTO(app, conv)
		if err != nil {
			return e.InternalServerError("Failed to load conversation", err)
		}

		dto := &conversationDetailDTO{conversationDTO: *base}

		if assigneeID := conv.GetString("assignee"); assigneeID != "" {
			if u, err := app.FindRecordById("users", assigneeID); err == nil {
				dto.AssigneeName = u.GetString("name")
			}
		}
		if teamID := conv.GetString("team"); teamID != "" {
			dto.TeamID = teamID
			if team, err := app.FindRecordById("teams", teamID); err == nil {
				dto.TeamName = team.GetString("name")
			}
		}

		inWindow, err := isWithinServiceWindow(app, convID)
		if err != nil {
			return e.InternalServerError("Failed to check service window", err)
		}
		dto.InWindow = inWindow

		return e.JSON(http.StatusOK, dto)
	}
}

// HandleConversationAssign assigns a conversation to a user.
func HandleConversationAssign(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}
		if !access.CanAssign() {
			return e.ForbiddenError("your role cannot assign conversations", nil)
		}

		convID := e.Request.PathValue("id")
		conv, err := app.FindRecordById("conversations", convID)
		if err != nil {
			return e.NotFoundError("conversation not found", nil)
		}
		if conv.GetString("org") != access.OrgID || !access.CanViewTeam(conv.GetString("team")) {
			return e.ForbiddenError("you don't have access to this conversation", nil)
		}

		var body struct {
			UserID     string `json:"user_id" form:"user_id"`
			RoundRobin bool   `json:"round_robin" form:"round_robin"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.BadRequestError("Invalid request body", err)
		}

		var target string
		if body.RoundRobin {
			assignee, err := store.AssignConversationRR(app, conv)
			if err != nil {
				return e.InternalServerError("Failed to pick a round-robin assignee", err)
			}
			target = assignee
		} else {
			if body.UserID != "" {
				// ensure the target user is a member of this org
				if _, err := store.FindOrgMembership(app, access.OrgID, body.UserID); err != nil {
					return e.BadRequestError("target user is not a member of this org", nil)
				}
				target = body.UserID
			}
			conv.Set("assignee", target)
			if err := app.Save(conv); err != nil {
				return e.InternalServerError("Failed to assign conversation", err)
			}
		}

		return e.JSON(http.StatusOK, map[string]any{"id": conv.Id, "assignee_id": target})
	}
}

// HandleConversationRead marks a conversation as read (resets unread counter).
func HandleConversationRead(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		convID := e.Request.PathValue("id")
		conv, err := app.FindRecordById("conversations", convID)
		if err != nil {
			return e.NotFoundError("conversation not found", nil)
		}
		if conv.GetString("org") != access.OrgID || !access.CanViewTeam(conv.GetString("team")) {
			return e.ForbiddenError("you don't have access to this conversation", nil)
		}

		if err := store.MarkConversationRead(app, convID); err != nil {
			return e.InternalServerError("Failed to mark conversation as read", err)
		}

		return e.JSON(http.StatusOK, map[string]any{"id": convID, "unread_count": 0})
	}
}
