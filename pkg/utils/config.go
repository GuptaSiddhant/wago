package utils

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

// AppConfig bundles the runtime configuration read from environment variables.
type AppConfig struct {
	AdminEmail            string
	AdminPassword         string
	WA_WebhookVerifyToken string
	MetaAppSecret         string

	// PublicBaseURL is the externally reachable base URL of this Wago instance
	// (e.g. https://wago.example.com). It is used to build the webhook callback
	// URL Meta delivers messages to. Leave empty if Wago is not publicly reachable.
	PublicBaseURL string

	SMTPHost        string
	SMTPPort        int
	SMTPUsername    string
	SMTPPassword    string
	SMTPTLS         bool
	SMTPFromAddress string
	SMTPFromName    string

	// VAPIDSubject is the contact (email or URL) sent in Web Push VAPID tokens
	// so push services can reach the operator. Defaults to AdminEmail.
	VAPIDSubject string

	// WA_NotificationTemplate is an approved Meta template name used to send
	// best-effort WhatsApp notifications to inactive users. Empty disables it.
	WA_NotificationTemplate string

	// Broadcast worker tuning. These bound how aggressively template broadcasts
	// are sent so the account stays within Meta/WhatsApp rate limits.
	MessagesPerMinute     int // global sustained rate across all active broadcasts
	BroadcastBatchSize    int // recipients claimed per broadcast per worker tick
	BroadcastLeaseSeconds int // how long a claimed recipient's lease lasts before redelivery
	BroadcastMaxAttempts  int // send attempts before a recipient is marked failed
}

// WorkerDefaults are applied when the corresponding env vars are unset.
const (
	DefaultMessagesPerMinute     = 60
	DefaultBroadcastBatchSize    = 10
	DefaultBroadcastLeaseSeconds = 300
	DefaultBroadcastMaxAttempts  = 3
)

// Helper to load environment variables
func LoadAppConfig() (*AppConfig, error) {
	_ = loadDotEnvFile(".env")

	adminEmail := getEnvOrDefault("ADMIN_EMAIL", "admin@wago.local")

	adminPassword := getRequiredEnv("ADMIN_PASSWORD")

	waWebhookVerifyToken := getRequiredEnv("WA_WEBHOOK_VERIFY_TOKEN")

	metaAppSecret := getEnvOrDefault("META_APP_SECRET", "")

	publicBaseURL := strings.TrimRight(getEnvOrDefault("PUBLIC_BASE_URL", ""), "/")

	smtpPort, _ := parseEnvInt("SMTP_PORT", 587)

	cfg := &AppConfig{
		AdminEmail:            adminEmail,
		AdminPassword:         adminPassword,
		WA_WebhookVerifyToken: waWebhookVerifyToken,
		MetaAppSecret:         metaAppSecret,
		PublicBaseURL:         publicBaseURL,

		SMTPHost:        os.Getenv("SMTP_HOST"),
		SMTPPort:        smtpPort,
		SMTPUsername:    os.Getenv("SMTP_USERNAME"),
		SMTPPassword:    os.Getenv("SMTP_PASSWORD"),
		SMTPTLS:         parseEnvBool("SMTP_TLS", false),
		SMTPFromAddress: getEnvOrDefault("SMTP_FROM_ADDRESS", adminEmail),
		SMTPFromName:    getEnvOrDefault("SMTP_FROM_NAME", "WaGo"),
		VAPIDSubject:    getEnvOrDefault("VAPID_SUBJECT", adminEmail),

		WA_NotificationTemplate: os.Getenv("WA_NOTIFICATION_TEMPLATE"),

		MessagesPerMinute:     parseEnvIntOrDefault("MESSAGES_PER_MINUTE", DefaultMessagesPerMinute),
		BroadcastBatchSize:    parseEnvIntOrDefault("BROADCAST_BATCH_SIZE", DefaultBroadcastBatchSize),
		BroadcastLeaseSeconds: parseEnvIntOrDefault("BROADCAST_LEASE_SECONDS", DefaultBroadcastLeaseSeconds),
		BroadcastMaxAttempts:  parseEnvIntOrDefault("BROADCAST_MAX_ATTEMPTS", DefaultBroadcastMaxAttempts),
	}

	return cfg, nil
}

// parseEnvIntOrDefault parses an int env var, falling back when empty/invalid.
func parseEnvIntOrDefault(key string, fallback int) int {
	val, err := parseEnvInt(key, fallback)
	if err != nil {
		return fallback
	}
	return val
}

// LoadDotEnv reads a .env file and sets the variables into the process env
func loadDotEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err // File doesn't exist
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on the first '=' sign
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		// Strip quotes if wrapped in "..." or '...'
		value = strings.Trim(value, `"'`)

		// Only set if not already set by system environment
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
	return scanner.Err()
}

// Returns the environment variable or a default string
func getRequiredEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("required env-var %s is missing", key)
	}
	return val
}

// Returns the environment variable or a default string
func getEnvOrDefault(key string, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// Parses a string env into a boolean with default fallback
func parseEnvBool(key string, fallback bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	// Accepts: "1", "t", "T", "true", "TRUE", "0", "f", "F", "false", "FALSE"
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return fallback
	}

	return parsed
}

// Parses a string env into an int with default fallback
func parseEnvInt(key string, fallback int) (int, error) {
	val := os.Getenv(key)
	if val == "" {
		return fallback, nil
	}
	return strconv.Atoi(val)
}
