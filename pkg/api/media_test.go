package api

import (
	"testing"

	"github.com/guptasiddhant/wago/pkg/meta"
)

func TestMediaKindForMime(t *testing.T) {
	tests := []struct {
		mime string
		want string
	}{
		{"image/jpeg", meta.KindImage},
		{"image/png", meta.KindImage},
		{"image/webp", meta.KindImage},
		{"video/mp4", meta.KindVideo},
		{"video/3gpp; codecs=avc1", meta.KindVideo},
		{"audio/ogg", meta.KindAudio},
		{"audio/mpeg", meta.KindAudio},
		{"application/pdf", meta.KindDocument},
		{"text/plain", meta.KindDocument},
		{"application/octet-stream", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := mediaKindForMime(tt.mime); got != tt.want {
			t.Errorf("mediaKindForMime(%q) = %q, want %q", tt.mime, got, tt.want)
		}
	}
}

func TestMediaCaptionText(t *testing.T) {
	tests := []struct {
		name     string
		kind     string
		caption  string
		filename string
		want     string
	}{
		{"caption wins", meta.KindImage, "check this out", "a.jpg", "check this out"},
		{"image label only", meta.KindImage, "", "", "[Image]"},
		{"image with filename", meta.KindImage, "", "a.jpg", "[Image] a.jpg"},
		{"audio", meta.KindAudio, "", "n.m4a", "[Audio] n.m4a"},
		{"document", meta.KindDocument, "", "r.pdf", "[Document] r.pdf"},
		{"video", meta.KindVideo, "clip", "clip.mp4", "clip"},
	}
	for _, tt := range tests {
		if got := mediaCaptionText(tt.kind, tt.caption, tt.filename); got != tt.want {
			t.Errorf("mediaCaptionText(%s) = %q, want %q", tt.name, got, tt.want)
		}
	}
}
