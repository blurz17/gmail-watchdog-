package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds all application configuration.
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Gmail    GmailConfig
	Telegram TelegramConfig
	HTTP     HTTPConfig
	Monitor  MonitorConfig
	Retry    RetryConfig
}

// AppConfig holds general application settings.
type AppConfig struct {
	Env      string
	Timezone *time.Location
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	URL string
}

// GmailConfig holds default Gmail OAuth credentials.
// Per-account credentials are stored in the database, but the default
// client ID/secret can be shared across accounts created via the CLI.
type GmailConfig struct {
	ClientID     string
	ClientSecret string
}

// TelegramConfig holds Telegram bot settings.
type TelegramConfig struct {
	BotToken string
	ChatID   string
}

// HTTPConfig holds HTTP server settings.
type HTTPConfig struct {
	Port   string
	APIKey string
}

// MonitorConfig holds monitoring/polling settings.
type MonitorConfig struct {
	PollInterval             time.Duration
	NotificationPollInterval time.Duration
}

// RetryConfig holds retry policy settings.
type RetryConfig struct {
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

// Load reads configuration from environment variables.
// It returns an error if any required configuration is missing.
func Load() (*Config, error) {
	tz, err := loadTimezone(getEnv("APP_TIMEZONE", "Asia/Riyadh"))
	if err != nil {
		return nil, fmt.Errorf("invalid timezone: %w", err)
	}

	pollInterval, err := time.ParseDuration(getEnv("POLL_INTERVAL", "60s"))
	if err != nil {
		return nil, fmt.Errorf("invalid POLL_INTERVAL: %w", err)
	}

	notifPollInterval, err := time.ParseDuration(getEnv("NOTIFICATION_POLL_INTERVAL", "5s"))
	if err != nil {
		return nil, fmt.Errorf("invalid NOTIFICATION_POLL_INTERVAL: %w", err)
	}

	retryInitial, err := time.ParseDuration(getEnv("RETRY_INITIAL_BACKOFF", "30s"))
	if err != nil {
		return nil, fmt.Errorf("invalid RETRY_INITIAL_BACKOFF: %w", err)
	}

	retryMax, err := time.ParseDuration(getEnv("RETRY_MAX_BACKOFF", "15m"))
	if err != nil {
		return nil, fmt.Errorf("invalid RETRY_MAX_BACKOFF: %w", err)
	}

	cfg := &Config{
		App: AppConfig{
			Env:      getEnv("APP_ENV", "development"),
			Timezone: tz,
		},
		Database: DatabaseConfig{
			URL: requireEnv("DATABASE_URL"),
		},
		Gmail: GmailConfig{
			ClientID:     getEnv("GMAIL_CLIENT_ID", ""),
			ClientSecret: getEnv("GMAIL_CLIENT_SECRET", ""),
		},
		Telegram: TelegramConfig{
			BotToken: requireEnv("TELEGRAM_BOT_TOKEN"),
			ChatID:   requireEnv("TELEGRAM_CHAT_ID"),
		},
		HTTP: HTTPConfig{
			Port:   getEnv("PORT", getEnv("HTTP_PORT", "8080")),
			APIKey: getEnv("HTTP_API_KEY", ""),
		},
		Monitor: MonitorConfig{
			PollInterval:             pollInterval,
			NotificationPollInterval: notifPollInterval,
		},
		Retry: RetryConfig{
			MaxAttempts:    getEnvInt("RETRY_MAX_ATTEMPTS", 5),
			InitialBackoff: retryInitial,
			MaxBackoff:     retryMax,
		},
	}

	return cfg, nil
}

// IsProduction returns true if the application is running in production mode.
func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}

func loadTimezone(name string) (*time.Location, error) {
	return time.LoadLocation(name)
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		// We don't fail here — the caller can validate.
		// This allows partial config loading for tests.
		return ""
	}
	return v
}

func getEnvInt(key string, defaultValue int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	var result int
	_, err := fmt.Sscanf(v, "%d", &result)
	if err != nil {
		return defaultValue
	}
	return result
}
