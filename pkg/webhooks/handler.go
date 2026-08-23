package webhooks

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/guptasiddhant/wago/pkg/meta"
	"github.com/guptasiddhant/wago/pkg/notifications"
	"github.com/guptasiddhant/wago/pkg/runtimecfg"
	"github.com/guptasiddhant/wago/pkg/store"

	"github.com/piusalfred/whatsapp/message"
	wawh "github.com/piusalfred/whatsapp/webhooks"
	"github.com/pocketbase/pocketbase/core"
)

// metaClient is used to download inbound media so it can be stored locally.
var metaClient = meta.NewClient()

// logf emits a structured log through PocketBase's slog logger when an app is
// available, falling back to the standard logger (e.g. in unit tests). The
// webhook path deliberately swallows errors to satisfy Meta's 200-expectation,
// so these logs are the only trace of processing problems.
func logf(app core.App, level slog.Level, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if app != nil {
		app.Logger().Log(context.Background(), level, msg,
			slog.String("component", "webhook"))
		return
	}
	slog.Log(context.Background(), level, msg, slog.String("component", "webhook"))
}

// HandleVerification returns the handler for GET /api/wa/webhook. The verify
// token is read live from the runtime config so Meta's handshake succeeds with
// whatever token the superadmin has configured.
func HandleVerification(mgr *runtimecfg.Manager) func(re *core.RequestEvent) error {
	return func(re *core.RequestEvent) error {
		cfg, err := mgr.Load(re.App)
		if err != nil {
			logf(re.App, slog.LevelWarn, "failed to load config: %v", err)
			return re.String(http.StatusForbidden, "Verification failed")
		}

		mode := re.Request.URL.Query().Get("hub.mode")
		token := re.Request.URL.Query().Get("hub.verify_token")
		challenge := re.Request.URL.Query().Get("hub.challenge")

		if mode == "subscribe" && token == cfg.WA_WebhookVerifyToken {
			logf(re.App, slog.LevelInfo, "Meta Webhook verified successfully")
			return re.String(http.StatusOK, challenge)
		}

		return re.String(http.StatusForbidden, "Verification failed")
	}
}

// HandleIncomingMessage returns the handler for POST /api/wa/webhook.
// When the current config has a non-empty Meta App Secret, inbound payloads are
// validated against the X-Hub-Signature-256 header signed with that secret. The
// notifier is triggered for assigned conversations so users get notified of new
// chats.
func HandleIncomingMessage(mgr *runtimecfg.Manager, notifier *notifications.Notifier) func(re *core.RequestEvent) error {
	return func(re *core.RequestEvent) error {
		cfg, err := mgr.Load(re.App)
		if err != nil {
			logf(re.App, slog.LevelWarn, "failed to load config: %v", err)
			return re.String(http.StatusOK, "EVENT_RECEIVED")
		}

		notification, err := wawh.ExtractAndValidatePayload(re.Request, &wawh.ValidateOptions{
			Validate:  cfg.MetaAppSecret != "",
			AppSecret: cfg.MetaAppSecret,
		})
		if err != nil {
			logf(re.App, slog.LevelWarn, "invalid webhook payload: %v", err)
			return re.String(http.StatusOK, "EVENT_RECEIVED")
		}

		processNotification(re.Request.Context(), re.App, notification, notifier)

		// Always return 200 OK to Meta immediately
		return re.String(http.StatusOK, "EVENT_RECEIVED")
	}
}

func processNotification(ctx context.Context, app core.App, notification *wawh.Notification, notifier *notifications.Notifier) {
	for _, entry := range notification.Entry {
		for _, change := range entry.Changes {
			processChange(ctx, app, change, notifier)
		}
	}
}

func processChange(ctx context.Context, app core.App, change wawh.Change, notifier *notifications.Notifier) {
	val := change.Value
	if val == nil || val.Metadata == nil {
		return
	}

	// route to the org owning this phone number
	account, err := store.FindWhatsAppAccountByPhoneNumberID(app, val.Metadata.PhoneNumberID)
	if err != nil {
		logf(app, slog.LevelWarn, "ignoring webhook for unknown phone_number_id %q", val.Metadata.PhoneNumberID)
		return
	}
	orgID := account.GetString("org")

	// delivery status updates (sent/delivered/read/failed)
	for _, st := range val.Statuses {
		if err := store.UpdateMessageStatus(app, st.ID, st.StatusValue); err != nil {
			logf(app, slog.LevelWarn, "failed to update message status %q -> %s: %v", st.ID, st.StatusValue, err)
		}
	}

	// incoming messages
	if len(val.Messages) == 0 {
		return
	}

	senderName := "Unknown"
	waID := ""
	if len(val.Contacts) > 0 {
		if val.Contacts[0].Profile != nil {
			senderName = val.Contacts[0].Profile.Name
		}
		waID = val.Contacts[0].WaID
	}
	if waID == "" {
		return
	}

	contact, err := store.UpsertContact(app, orgID, waID, senderName)
	if err != nil {
		logf(app, slog.LevelWarn, "failed to upsert contact: %v", err)
		return
	}

	ts := time.Now()
	conv, created, err := store.UpsertConversation(app, orgID, contact.Id, account.Id, ts)
	if err != nil {
		logf(app, slog.LevelWarn, "failed to upsert conversation: %v", err)
		return
	}

	// New conversations are auto-assigned round-robin to an eligible agent.
	if created {
		if assignee, err := store.AssignConversationRR(app, conv); err != nil {
			logf(app, slog.LevelWarn, "failed to round-robin assign conversation %s: %v", conv.Id, err)
		} else if assignee != "" {
			logf(app, slog.LevelInfo, "round-robin assigned conversation %s to %s", conv.Id, assignee)
		}
	}

	for _, msg := range val.Messages {
		if msg == nil {
			continue
		}

		body := messageBody(msg)
		payload, mediaInfo := inboundPayload(msg)

		msgTS := ts
		if sec, err := strconv.ParseInt(msg.Timestamp, 10, 64); err == nil {
			msgTS = time.Unix(sec, 0)
		}

		// Download and store media bytes locally so the inbox can preview and
		// download the file without hitting Meta again.
		var mediaData []byte
		var mediaName string
		if mediaInfo != nil && mediaInfo.ID != "" {
			token := account.GetString("access_token")
			if token == "" {
				logf(app, slog.LevelWarn, "cannot download media for message %s: account has no access token", msg.ID)
			} else {
				data, err := downloadInboundMedia(ctx, app, token, mediaInfo)
				if err != nil {
					logf(app, slog.LevelWarn, "failed to download media for message %s: %v", msg.ID, err)
				} else {
					mediaData = data
					mediaName = mediaFilename(mediaInfo)
				}
			}
		}

		err := store.SaveIncomingMessage(
			app, orgID, conv.Id, msg.From, senderName,
			val.Metadata.DisplayPhoneNumber, body, msg.ID, msgTS, payload,
			mediaData, mediaName,
		)
		if err != nil {
			logf(app, slog.LevelError, "failed to save message: %v", err)
			continue
		}

		// inbound messages count towards unread until an agent reads them
		_ = store.IncrementConversationUnread(app, conv.Id)

		// Notify the assigned agent (desktop push if active, email/WhatsApp if not).
		notifier.Trigger(ctx, app, orgID, conv.Id, conv.GetString("assignee"), body)
	}
}

// inboundPayload normalizes a received message into the stored JSON payload so
// callers can tell the message kind and its media metadata apart from the body
// text. It returns the payload plus the raw media info (nil for non-media
// messages) so the caller can download the attachment.
func inboundPayload(m *wawh.Message) (map[string]any, *message.MediaInfo) {
	if m == nil {
		return map[string]any{"type": "text"}, nil
	}

	switch {
	case m.Text != nil:
		return map[string]any{"type": "text"}, nil
	case m.Image != nil:
		return mediaPayloadOf(meta.KindImage, m.Image, m.Image.Caption), m.Image
	case m.Video != nil:
		return mediaPayloadOf(meta.KindVideo, m.Video, m.Video.Caption), m.Video
	case m.Audio != nil:
		return mediaPayloadOf(meta.KindAudio, m.Audio, ""), m.Audio
	case m.Document != nil:
		return mediaPayloadOf(meta.KindDocument, m.Document, m.Document.Caption), m.Document
	case m.Sticker != nil:
		return mediaPayloadOf(meta.KindSticker, m.Sticker, ""), m.Sticker
	default:
		return map[string]any{"type": "text"}, nil
	}
}

type mediaPayload struct {
	MediaID  string `json:"media_id,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Filename string `json:"filename,omitempty"`
	Caption  string `json:"caption,omitempty"`
}

func mediaPayloadOf(kind string, info *message.MediaInfo, caption string) map[string]any {
	if info == nil {
		return map[string]any{"type": kind}
	}
	return map[string]any{
		"type": kind,
		"media": mediaPayload{
			MediaID:  info.ID,
			MimeType: info.MimeType,
			Filename: info.Filename,
			Caption:  caption,
		},
	}
}

// downloadInboundMedia retrieves the file bytes for a received media message
// from Meta so they can be stored in PocketBase storage.
func downloadInboundMedia(ctx context.Context, app core.App, accessToken string, info *message.MediaInfo) ([]byte, error) {
	// Add timeout for media download
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	url, _, err := metaClient.GetMediaRetrieve(ctx, accessToken, info.ID)
	if err != nil {
		return nil, err
	}
	return metaClient.DownloadMedia(ctx, url, accessToken)
}

// mediaFilename returns a stable, extension-bearing filename for a stored media
// file derived from the Meta media id and MIME type.
func mediaFilename(info *message.MediaInfo) string {
	name := info.Filename
	if name == "" {
		name = "media"
	}
	ext := extensionForMime(info.MimeType)
	if ext == "" {
		ext = path.Ext(name)
	}
	return strings.TrimSuffix(name, path.Ext(name)) + ext
}

func extensionForMime(mimeType string) string {
	if i := strings.Index(mimeType, ";"); i >= 0 {
		mimeType = strings.TrimSpace(mimeType[:i])
	}
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "video/mp4":
		return ".mp4"
	case "video/3gpp":
		return ".3gp"
	case "audio/ogg":
		return ".ogg"
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/amr":
		return ".amr"
	case "audio/opus":
		return ".opus"
	case "application/pdf":
		return ".pdf"
	default:
		if ext := path.Ext(mimeType); ext != "" {
			return ext
		}
		return ""
	}
}

// messageBody renders a human-readable body for any inbound message type so
// media, buttons, interactive replies, locations and other non-text messages
// are shown (and notified on) instead of being silently dropped.
func messageBody(m *wawh.Message) string {
	switch {
	case m == nil:
		return ""
	case m.Text != nil:
		return m.Text.Body
	case m.Image != nil:
		return mediaBody("Image", m.Image)
	case m.Video != nil:
		return mediaBody("Video", m.Video)
	case m.Audio != nil:
		return mediaBody("Audio", m.Audio)
	case m.Document != nil:
		return mediaBody("Document", m.Document)
	case m.Sticker != nil:
		return mediaBody("Sticker", m.Sticker)
	case m.Button != nil:
		if m.Button.Text != "" {
			return "Button: " + m.Button.Text
		}
		if m.Button.Payload != "" {
			return "Button: " + m.Button.Payload
		}
		return "[Button]"
	case m.Interactive != nil:
		if r := m.Interactive.ButtonReply; r != nil {
			if r.Title != "" {
				return "Button reply: " + r.Title
			}
			return "Button reply: " + r.ID
		}
		if r := m.Interactive.ListReply; r != nil {
			if r.Title != "" {
				return "List reply: " + r.Title
			}
			return "List reply: " + r.ID
		}
		if r := m.Interactive.NFMReply; r != nil {
			if r.Body != "" {
				return "Flow reply: " + r.Body
			}
			return "[Flow reply]"
		}
		return "[Interactive: " + m.Interactive.Type + "]"
	case m.Location != nil:
		loc := fmt.Sprintf("Location: %.5f, %.5f", m.Location.Latitude, m.Location.Longitude)
		if m.Location.Name != "" {
			loc += " (" + m.Location.Name + ")"
		}
		return loc
	case m.Contacts != nil && len(*m.Contacts) > 0:
		if name := contactName((*m.Contacts)[0]); name != "" {
			return "Contact: " + name
		}
		return "[Contact]"
	case m.System != nil:
		return "System: " + m.System.Body
	case m.Reaction != nil:
		if m.Reaction.Emoji != "" {
			return "Reaction: " + m.Reaction.Emoji
		}
		return "[Reaction]"
	default:
		// Fall back to a bracketed type so nothing is lost silently.
		return "[" + m.Type + "]"
	}
}

func mediaBody(kind string, media *message.MediaInfo) string {
	s := "[" + kind + "]"
	if media.Caption != "" {
		s += " " + media.Caption
	} else if media.Filename != "" {
		s += " " + media.Filename
	}
	return s
}

func contactName(c *message.Contact) string {
	if c == nil || c.Name == nil {
		return ""
	}
	if c.Name.FormattedName != "" {
		return c.Name.FormattedName
	}
	return strings.TrimSpace(c.Name.FirstName + " " + c.Name.LastName)
}
