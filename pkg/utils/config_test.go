package utils

import (
	"strings"
	"testing"
)

func validConfig() *AppConfig {
	return &AppConfig{
		MessagesPerMinute:     60,
		BroadcastBatchSize:    10,
		BroadcastLeaseSeconds: 300,
		BroadcastMaxAttempts:  3,
	}
}

func TestValidateAcceptsEmptyOptionalConfig(t *testing.T) {
	cfg := validConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected defaults-only config to be valid, got %v", err)
	}
}

func TestValidateRejectsBadBaseURL(t *testing.T) {
	cfg := validConfig()
	cfg.PublicBaseURL = "wago.example.com"
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "public_base_url") {
		t.Fatalf("expected public_base_url error, got %v", err)
	}
}

func TestValidateRequiresAIFieldsWhenEnabled(t *testing.T) {
	cfg := validConfig()
	cfg.AIEnabled = true
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected ai_base_url and ai_model errors when AI enabled without config")
	}
	if !strings.Contains(err.Error(), "ai_base_url") || !strings.Contains(err.Error(), "ai_model") {
		t.Fatalf("expected both AI field errors, got %v", err)
	}

	cfg.AIBaseURL = "https://api.openai.com/v1"
	cfg.AIModel = "gpt-4o-mini"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid after filling AI fields, got %v", err)
	}
}

func TestValidateRejectsBadSMTPPort(t *testing.T) {
	cfg := validConfig()
	cfg.SMTPHost = "smtp.example.com"
	cfg.SMTPPort = 0
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "smtp_port") {
		t.Fatalf("expected smtp_port error, got %v", err)
	}
}

func TestValidateRejectsNonPositiveWorkerTuning(t *testing.T) {
	cfg := validConfig()
	cfg.MessagesPerMinute = 0
	cfg.BroadcastBatchSize = -1
	cfg.BroadcastLeaseSeconds = 0
	cfg.BroadcastMaxAttempts = -3
	err := cfg.Validate()
	for _, want := range []string{
		"messages_per_minute",
		"broadcast_batch_size",
		"broadcast_lease_seconds",
		"broadcast_max_attempts",
	} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error mentioning %s, got %v", want, err)
		}
	}
}
