package api

import (
	"reflect"
	"testing"

	"github.com/guptasiddhant/wago/pkg/meta"
)

func TestCountVariables(t *testing.T) {
	tests := []struct {
		body string
		want int
	}{
		{"hello", 0},
		{"hi {{1}}", 1},
		{"hi {{1}} and {{3}}", 3},
		{"{{2}}{{1}}", 2},
		{"{{10}}", 10},
	}
	for _, tt := range tests {
		if got := countVariables(tt.body); got != tt.want {
			t.Errorf("countVariables(%q) = %d, want %d", tt.body, got, tt.want)
		}
	}
}

func TestExampleBodyText(t *testing.T) {
	// No variables -> no example payload.
	if got := exampleBodyText("no vars", nil); got != nil {
		t.Errorf("expected nil example for variable-free body, got %v", got)
	}

	// Uses provided values in order.
	got := exampleBodyText("hi {{1}} {{2}}", []string{"Alice", "Bob"})
	rows, ok := got["body_text"].([][]string)
	if !ok {
		t.Fatalf("body_text wrong type: %T", got["body_text"])
	}
	if len(rows) != 1 || len(rows[0]) != 2 {
		t.Fatalf("unexpected body_text shape: %v", rows)
	}
	if rows[0][0] != "Alice" || rows[0][1] != "Bob" {
		t.Errorf("expected [Alice Bob], got %v", rows[0])
	}

	// Missing values fall back to "Sample".
	fallback := exampleBodyText("hi {{1}}", []string{""})
	rows2 := fallback["body_text"].([][]string)
	if rows2[0][0] != "Sample" {
		t.Errorf("expected Sample fallback, got %q", rows2[0][0])
	}
}

func TestBuildTemplateSubmissionMediaHeader(t *testing.T) {
	body := templateCreateRequest{
		Name:            "sale",
		Language:        "en_US",
		Category:        "MARKETING",
		HeaderType:      "MEDIA",
		HeaderMediaType: "IMAGE",
		HeaderMediaID:   "12345",
		Body:            "Flash sale {{1}}",
	}

	sub := buildTemplateSubmission(body)
	if sub.Name != "sale" {
		t.Fatalf("unexpected name %q", sub.Name)
	}

	var header *meta.TemplateComponent
	for _, c := range sub.Components {
		if c.Type == "HEADER" {
			header = &c
		}
	}
	if header == nil {
		t.Fatal("expected a HEADER component for a media template")
	}
	if header.Format != "IMAGE" {
		t.Errorf("expected IMAGE format, got %q", header.Format)
	}
	handles, ok := header.Example["header_handle"].([]string)
	if !ok || len(handles) != 1 || handles[0] != "12345" {
		t.Errorf("expected header_handle [12345], got %v", header.Example["header_handle"])
	}

	// DOCUMENT headers carry format but no example handle.
	docBody := body
	docBody.HeaderMediaType = "DOCUMENT"
	docSub := buildTemplateSubmission(docBody)
	for _, c := range docSub.Components {
		if c.Type == "HEADER" {
			if c.Format != "DOCUMENT" {
				t.Errorf("expected DOCUMENT format, got %q", c.Format)
			}
			if _, ok := c.Example["header_handle"]; ok {
				t.Errorf("DOCUMENT headers should not include a header_handle example")
			}
		}
	}
}

func TestBuildTemplateSubmissionTextHeaderUntouched(t *testing.T) {
	body := templateCreateRequest{
		HeaderType: "TEXT",
		HeaderText: "Hey",
		Body:       "world",
	}
	sub := buildTemplateSubmission(body)
	for _, c := range sub.Components {
		if c.Type == "HEADER" {
			if c.Format != "" {
				t.Errorf("text header should have no format, got %q", c.Format)
			}
			if c.Text != "Hey" {
				t.Errorf("expected header text, got %q", c.Text)
			}
		}
	}
}

func TestBuildTemplateSubmissionComponentsOrder(t *testing.T) {
	body := templateCreateRequest{
		HeaderType:      "MEDIA",
		HeaderMediaType: "VIDEO",
		HeaderMediaID:   "vid1",
		Body:            "hello",
		Footer:          "bye",
	}
	sub := buildTemplateSubmission(body)
	var order []string
	for _, c := range sub.Components {
		order = append(order, c.Type)
	}
	want := []string{"HEADER", "BODY", "FOOTER"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("component order = %v, want %v", order, want)
	}
}
