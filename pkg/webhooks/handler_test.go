package webhooks

import (
	"testing"

	"github.com/piusalfred/whatsapp/message"
	wawh "github.com/piusalfred/whatsapp/webhooks"
)

func TestMessageBody(t *testing.T) {
	tests := []struct {
		name string
		msg  *wawh.Message
		want string
	}{
		{"nil", nil, ""},
		{"text", &wawh.Message{Type: "text", Text: &wawh.Text{Body: "hello"}}, "hello"},
		{"image with caption", &wawh.Message{Type: "image", Image: &message.MediaInfo{Caption: "the goods"}}, "[Image] the goods"},
		{"document with filename", &wawh.Message{Type: "document", Document: &message.MediaInfo{Filename: "report.pdf"}}, "[Document] report.pdf"},
		{"video no caption", &wawh.Message{Type: "video", Video: &message.MediaInfo{}}, "[Video]"},
		{"sticker", &wawh.Message{Type: "sticker", Sticker: &message.MediaInfo{}}, "[Sticker]"},
		{"button text", &wawh.Message{Type: "button", Button: &wawh.Button{Text: "Yes please"}}, "Button: Yes please"},
		{"button payload only", &wawh.Message{Type: "button", Button: &wawh.Button{Payload: "order-1"}}, "Button: order-1"},
		{"interactive button reply", &wawh.Message{Type: "interactive", Interactive: &wawh.Interactive{ButtonReply: &wawh.ButtonReply{Title: "I'm interested"}}}, "Button reply: I'm interested"},
		{"interactive list reply", &wawh.Message{Type: "interactive", Interactive: &wawh.Interactive{ListReply: &wawh.ListReply{Title: "Option B"}}}, "List reply: Option B"},
		{"interactive flow reply", &wawh.Message{Type: "interactive", Interactive: &wawh.Interactive{NFMReply: &wawh.NFMReply{Body: "john@example.com"}}}, "Flow reply: john@example.com"},
		{"interactive unknown type", &wawh.Message{Type: "interactive", Interactive: &wawh.Interactive{Type: "flow"}}, "[Interactive: flow]"},
		{"location", &wawh.Message{Type: "location", Location: &message.Location{Latitude: 37.7749, Longitude: -122.4194, Name: "SF"}}, "Location: 37.77490, -122.41940 (SF)"},
		{"contact", &wawh.Message{Type: "contacts", Contacts: &message.Contacts{&message.Contact{Name: &message.Name{FormattedName: "Jane Doe"}}}}, "Contact: Jane Doe"},
		{"system", &wawh.Message{Type: "system", System: &wawh.System{Body: "started chatting"}}, "System: started chatting"},
		{"reaction", &wawh.Message{Type: "reaction", Reaction: &message.Reaction{Emoji: "❤️"}}, "Reaction: ❤️"},
		{"template with text", &wawh.Message{Type: "template", Text: &wawh.Text{Body: "template text"}}, "template text"},
		{"unknown", &wawh.Message{Type: "unknown"}, "[unknown]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := messageBody(tt.msg); got != tt.want {
				t.Errorf("messageBody(%s) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestInboundPayload(t *testing.T) {
	tests := []struct {
		name      string
		msg       *wawh.Message
		wantType  string
		wantID    string
		wantMedia bool
	}{
		{"nil", nil, "text", "", false},
		{"text", &wawh.Message{Type: "text", Text: &wawh.Text{Body: "hi"}}, "text", "", false},
		{"image", &wawh.Message{Type: "image", Image: &message.MediaInfo{ID: "mid-1", MimeType: "image/jpeg", Filename: "a.jpg", Caption: "c"}}, "image", "mid-1", true},
		{"video", &wawh.Message{Type: "video", Video: &message.MediaInfo{ID: "vid-2"}}, "video", "vid-2", true},
		{"audio", &wawh.Message{Type: "audio", Audio: &message.MediaInfo{ID: "aud-3"}}, "audio", "aud-3", true},
		{"document", &wawh.Message{Type: "document", Document: &message.MediaInfo{ID: "doc-4", Filename: "r.pdf"}}, "document", "doc-4", true},
		{"sticker", &wawh.Message{Type: "sticker", Sticker: &message.MediaInfo{ID: "stk-5"}}, "sticker", "stk-5", true},
		{"button falls back to text", &wawh.Message{Type: "button", Button: &wawh.Button{Text: "ok"}}, "text", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, info := inboundPayload(tt.msg)
			if payload["type"] != tt.wantType {
				t.Errorf("payload type = %v, want %v", payload["type"], tt.wantType)
			}
			if tt.wantMedia && info == nil {
				t.Fatal("expected media info, got nil")
			}
			if info != nil && info.ID != tt.wantID {
				t.Errorf("media id = %q, want %q", info.ID, tt.wantID)
			}
			if !tt.wantMedia && info != nil {
				t.Errorf("did not expect media info, got %+v", info)
			}
		})
	}
}

func TestMediaFilename(t *testing.T) {
	tests := []struct {
		name string
		info *message.MediaInfo
		want string
	}{
		{"keeps existing filename", &message.MediaInfo{ID: "m1", Filename: "photo.jpg", MimeType: "image/jpeg"}, "photo.jpg"},
		{"derives from mime", &message.MediaInfo{ID: "m2", MimeType: "application/pdf"}, "media.pdf"},
		{"video 3gpp", &message.MediaInfo{ID: "m3", MimeType: "video/3gpp"}, "media.3gp"},
		{"unknown mime keeps ext", &message.MediaInfo{ID: "m4", Filename: "archive.7z", MimeType: "application/octet-stream"}, "archive.7z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mediaFilename(tt.info); got != tt.want {
				t.Errorf("mediaFilename(%s) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestExtensionForMime(t *testing.T) {
	tests := []struct {
		mime string
		want string
	}{
		{"image/jpeg", ".jpg"},
		{"image/png", ".png"},
		{"image/webp", ".webp"},
		{"video/mp4", ".mp4"},
		{"video/3gpp", ".3gp"},
		{"audio/mpeg", ".mp3"},
		{"application/pdf", ".pdf"},
		{"image/jpeg; charset=binary", ".jpg"},
		{"application/octet-stream", ""},
	}
	for _, tt := range tests {
		if got := extensionForMime(tt.mime); got != tt.want {
			t.Errorf("extensionForMime(%q) = %q, want %q", tt.mime, got, tt.want)
		}
	}
}
