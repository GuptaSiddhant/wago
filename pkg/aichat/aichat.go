package aichat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Config configures an OpenAI-compatible chat-completions provider. Wago stays
// model-agnostic: any endpoint speaking the OpenAI wire format works (OpenAI,
// DeepSeek, Ollama, vLLM, LiteLLM, ...).
type Config struct {
	Enabled bool
	BaseURL string // e.g. https://api.openai.com/v1 (no trailing slash)
	APIKey  string
	Model   string
}

// NewClient builds an aichat client from config.
func NewClient(cfg Config) *Client {
	return &Client{cfg: cfg}
}

// Client streams assistant replies from a configured provider.
type Client struct {
	cfg Config
}

// Message is a chat message in the OpenAI wire format.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Enabled reports whether the AI feature is turned on.
func (c *Client) Enabled() bool { return c != nil && c.cfg.Enabled }

// DeltaHandler is invoked for each incremental assistant text token produced
// by the provider.
type DeltaHandler func(delta string)

// ErrorHandler is invoked with a non-nil error if the call fails or the
// stream ends unexpectedly.
type ErrorHandler func(err error)

// Completion streams a chat completion. It builds the OpenAI-compatible
// request, parses the provider's SSE response, and calls onDelta for each
// text token. The onError handler receives fatal errors (request construction,
// HTTP failure, malformed stream).
func (c *Client) Completion(ctx context.Context, messages []Message, onDelta DeltaHandler, onError ErrorHandler) {
	if !c.Enabled() {
		onError(fmt.Errorf("AI is not enabled"))
		return
	}
	if c.cfg.BaseURL == "" || c.cfg.Model == "" {
		onError(fmt.Errorf("AI provider is not configured (AI_BASE_URL / AI_MODEL)"))
		return
	}

	body, err := json.Marshal(map[string]any{
		"model":    c.cfg.Model,
		"messages": messages,
		"stream":   true,
	})
	if err != nil {
		onError(fmt.Errorf("aichat: marshal request: %w", err))
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(c.cfg.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		onError(fmt.Errorf("aichat: build request: %w", err))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		onError(fmt.Errorf("aichat: request failed: %w", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		onError(fmt.Errorf("aichat: provider returned %s: %s", resp.Status, strings.TrimSpace(string(msg))))
		return
	}

	if err := c.streamLines(ctx, resp.Body, onDelta); err != nil {
		onError(err)
	}
}

// streamLines parses provider SSE `data:` lines, extracting choices[0].delta.
// Returns nil when the stream terminates normally ([DONE] sentinel or EOF).
func (c *Client) streamLines(ctx context.Context, body io.Reader, onDelta DeltaHandler) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return nil
		}
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			return nil
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			// Some providers emit keep-alive lines; skip anything we can't parse.
			continue
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			onDelta(chunk.Choices[0].Delta.Content)
		}
	}
	return scanner.Err()
}
