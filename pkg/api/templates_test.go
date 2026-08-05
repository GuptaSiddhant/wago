package api

import (
	"testing"
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
