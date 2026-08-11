package aichat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompletionStreamsDeltas(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Errorf("authorization = %q, want Bearer sk-test", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		chunks := []string{
			`data: {"choices":[{"delta":{"content":"Hel"}}]}`,
			`data: {"choices":[{"delta":{"content":"lo "}}]}`,
			`data: {"choices":[{"delta":{"content":"world"}}]}`,
			`data: [DONE]`,
		}
		for _, c := range chunks {
			if _, err := w.Write([]byte(c + "\n\n")); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	client := NewClient(Config{
		Enabled: true,
		BaseURL: srv.URL + "/v1",
		APIKey:  "sk-test",
		Model:   "gpt-4o-mini",
	})

	var got strings.Builder
	var gotErr error
	client.Completion(context.Background(), []Message{
		{Role: "user", Content: "hi"},
	}, func(d string) {
		got.WriteString(d)
	}, func(err error) {
		gotErr = err
	})

	if gotErr != nil {
		t.Fatalf("unexpected error: %v", gotErr)
	}
	if got.String() != "Hello world" {
		t.Errorf("collected %q, want %q", got.String(), "Hello world")
	}
}

func TestCompletionNotEnabled(t *testing.T) {
	client := NewClient(Config{Enabled: false})
	var gotErr error
	client.Completion(context.Background(), nil, func(string) {}, func(err error) {
		gotErr = err
	})
	if gotErr == nil {
		t.Fatal("expected error when AI is disabled")
	}
}

func TestCompletionProviderError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"bad key"}}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := NewClient(Config{Enabled: true, BaseURL: srv.URL + "/v1", APIKey: "sk", Model: "m"})
	var gotErr error
	client.Completion(context.Background(), nil, func(string) {}, func(err error) {
		gotErr = err
	})
	if gotErr == nil {
		t.Fatal("expected provider error")
	}
	if !strings.Contains(gotErr.Error(), "bad key") {
		t.Errorf("error = %q, want it to include the provider message", gotErr)
	}
}
