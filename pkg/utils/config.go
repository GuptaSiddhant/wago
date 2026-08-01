package utils

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

type AppConfig struct {
	AdminEmail            string
	AdminPassword         string
	WA_WebhookVerifyToken string
	Headless              bool
}

// Helper to load environment variables
func LoadAppConfig() (*AppConfig, error) {
	// _ = loadDotEnvFile(".env.local")
	_ = loadDotEnvFile(".env")

	adminEmail := getEnvOrDefault("ADMIN_EMAIL", "admin@wago.local")

	adminPassword := getRequiredEnv("ADMIN_PASSWORD")

	waWebhookVerifyToken := getRequiredEnv("WA_WEBHOOK_VERIFY_TOKEN")

	headless := parseEnvBool("HEADLESS", false)

	cfg := &AppConfig{
		AdminEmail:            adminEmail,
		AdminPassword:         adminPassword,
		WA_WebhookVerifyToken: waWebhookVerifyToken,
		Headless:              headless,
	}

	return cfg, nil
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

// Splits a comma-separated string env into a slice
func parseEnvSlice(key string, fallback []string) []string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parts := strings.Split(val, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}
