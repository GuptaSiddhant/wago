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
