package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/guptasiddhant/wago/pkg/aichat"
	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/pocketbase/pocketbase/core"
)

// aiCfg is the AI provider configuration set on Register and read by the
// chat handler.
var aiCfg = aichat.Config{}

// aiChatMessage is the message shape the TanStack AI client sends over the
// wire (UIMessage). Only the text parts are meaningful to the assistant.
type aiChatMessage struct {
	Role  string `json:"role"`
	Parts []struct {
		Type    string `json:"type"`
		Content string `json:"content"`
	} `json:"parts"`
}

// aiChatRequest is the AG-UI chat request payload produced by useChat via a
// fetchServerSentEvents connection.
type aiChatRequest struct {
	Messages       []aiChatMessage `json:"messages"`
	Data           map[string]any  `json:"data"`
	ForwardedProps map[string]any  `json:"forwardedProps"`
	ConversationID string          `json:"conversationId"`
}

// HandleAIChat streams assistant replies for the home dashboard chat. When the
// request references a conversation, its recent transcript is injected as
// context so the bot can summarize and answer questions about it. Replies are
// emitted in the TanStack AI (AG-UI) SSE wire format:
//
//	data: {"type":"RUN_STARTED",...}\n\n
//	data: {"type":"TEXT_MESSAGE_CONTENT","delta":"..."}\n\n
//	...
//	data: [DONE]
//
// The endpoint is only reachable when AI is enabled in the backend config.
func HandleAIChat(app core.App, ai *aichat.Client) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if !ai.Enabled() {
			return e.ForbiddenError("the AI assistant is not enabled on this instance", nil)
		}

		access, apiErr := orgAccessFromRequest(e, app)
		if apiErr != nil {
			return apiErr
		}

		var req aiChatRequest
		if err := e.BindBody(&req); err != nil {
			return e.BadRequestError("Invalid request body", err)
		}

		convID := strings.TrimSpace(req.ConversationID)
		if convID == "" {
			convID = strFromAny(req.Data["conversationId"])
		}
		if convID == "" {
			convID = strFromAny(req.ForwardedProps["conversationId"])
		}

		// Resolve the conversation context up front so auth failures surface as
		// a clean SSE error rather than after the stream has started.
		system, err := aiSystemPrompt(e, app, access, convID)
		if err != nil {
			return err
		}

		messages := make([]aichat.Message, 0, len(req.Messages)+1)
		if system != "" {
			messages = append(messages, aichat.Message{Role: "system", Content: system})
		}
		for _, m := range req.Messages {
			content := ""
			for _, p := range m.Parts {
				if p.Type == "text" {
					content += p.Content
				}
			}
			if strings.TrimSpace(content) == "" {
				continue
			}
			messages = append(messages, aichat.Message{Role: m.Role, Content: content})
		}

		e.Response.Header().Set("Content-Type", "text/event-stream")
		e.Response.Header().Set("Cache-Control", "no-cache")
		e.Response.Header().Set("X-Accel-Buffering", "no")
		rc := http.NewResponseController(e.Response)
		rc.SetWriteDeadline(time.Time{})
		ctx := e.Request.Context()

		runID := newID()
		msgID := newID()

		writeChunk := func(payload string) error {
			if _, err := e.Response.Write([]byte("data: " + payload + "\n\n")); err != nil {
				return err
			}
			return rc.Flush()
		}
		writeEvent := func(v any) error {
			b, err := json.Marshal(v)
			if err != nil {
				return err
			}
			return writeChunk(string(b))
		}

		sendError := func(err error) {
			_ = writeEvent(map[string]any{
				"type":      "RUN_ERROR",
				"timestamp": nowMillis(),
				"error":     map[string]any{"message": err.Error()},
			})
			_, _ = e.Response.Write([]byte("data: [DONE]\n\n"))
			_ = rc.Flush()
		}

		if err := writeEvent(map[string]any{
			"type":      "RUN_STARTED",
			"runId":     runID,
			"timestamp": nowMillis(),
		}); err != nil {
			return nil
		}

		started := false
		streamErrored := false
		ai.Completion(ctx, messages,
			func(delta string) {
				if err := ctx.Err(); err != nil {
					return
				}
				if !started {
					started = true
					_ = writeEvent(map[string]any{
						"type":      "TEXT_MESSAGE_START",
						"messageId": msgID,
						"role":      "assistant",
						"timestamp": nowMillis(),
					})
				}
				_ = writeEvent(map[string]any{
					"type":      "TEXT_MESSAGE_CONTENT",
					"messageId": msgID,
					"delta":     delta,
					"timestamp": nowMillis(),
				})
			},
			func(err error) {
				if err == nil {
					return
				}
				if ctx.Err() != nil {
					return // client disconnected; skip error handling
				}
				streamErrored = true
				sendError(err)
			})

		if ctx.Err() != nil || streamErrored {
			// sendError already emitted RUN_ERROR + [DONE].
			return nil
		}
		if started {
			_ = writeEvent(map[string]any{
				"type":      "TEXT_MESSAGE_END",
				"messageId": msgID,
				"timestamp": nowMillis(),
			})
		}
		_ = writeEvent(map[string]any{
			"type":         "RUN_FINISHED",
			"runId":        runID,
			"finishReason": "stop",
			"timestamp":    nowMillis(),
		})
		_, _ = e.Response.Write([]byte("data: [DONE]\n\n"))
		return rc.Flush()
	}
}

// aiSystemPrompt builds the assistant's system message. When a conversation is
// referenced it includes the contact context and the recent transcript so the
// bot can summarize and reason about it.
func aiSystemPrompt(e *core.RequestEvent, app core.App, access *store.OrgAccess, convID string) (string, error) {
	if convID == "" {
		return "You are the WaGo assistant, a helpful support agent for a WhatsApp helpdesk. " +
			"Answer questions about the user's conversations when context is provided. Be concise and practical.", nil
	}

	conv, err := store.FindOrgRecord(app, access.OrgID, "conversations", convID)
	if err != nil {
		return "", e.NotFoundError("conversation not found", nil)
	}
	if !access.CanViewTeam(conv.GetString("team")) {
		return "", e.ForbiddenError("you don't have access to this conversation", nil)
	}

	contact := "unknown"
	if c, err := app.FindRecordById("contacts", conv.GetString("contact")); err == nil {
		contact = c.GetString("name")
		if phone := c.GetString("phone"); phone != "" {
			if contact != "" {
				contact += " (" + phone + ")"
			} else {
				contact = phone
			}
		}
	}

	records, err := app.FindRecordsByFilter("messages",
		"conversation = {:conv}", "-created", 50, 0,
		store.DbxParams(map[string]any{"conv": convID}))
	if err != nil {
		return "", e.InternalServerError("Failed to load transcript", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "You are the WaGo assistant for a WhatsApp helpdesk. "+
		"The user selected a conversation with %s.\n\n", contact)
	b.WriteString("Here is the recent transcript (oldest first, lines prefixed with the sender and the time):\n")
	for i := len(records) - 1; i >= 0; i-- {
		msg := records[i]
		role := "contact"
		if msg.GetString("direction") == "outbound" {
			role = "agent"
		}
		when := msg.GetDateTime("created").Time().UTC().Format("Jan 2 15:04")
		body := strings.TrimSpace(msg.GetString("body"))
		if body == "" {
			body = "[media message]"
		}
		fmt.Fprintf(&b, "%s (%s): %s\n", role, when, body)
	}
	b.WriteString("\nAnswer the user's question, optionally summarizing the conversation " +
		"or drafting a reply on their behalf. Be concise and practical.")
	return b.String(), nil
}

func strFromAny(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func newID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func nowMillis() int64 {
	return time.Now().UnixMilli()
}
