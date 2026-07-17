package app

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	App         AppConfig
	HTTP        HTTPConfig
	Database    DatabaseConfig
	Log         LogConfig
	Google      GoogleConfig
	Auth        AuthConfig
	Storage     StorageConfig
	Fulfillment FulfillmentConfig
	Payments    PaymentsConfig
}

type AppConfig struct {
	Name string
	Env  string
}

type HTTPConfig struct {
	Addr string
}

type DatabaseConfig struct {
	URL string
}

type LogConfig struct {
	Level  string
	Format string
}

type GoogleConfig struct {
	ClientID string
}

type AuthConfig struct {
	SigningSecret string
	SessionTTL    time.Duration
}

type StorageConfig struct {
	Endpoint        string
	Bucket          string
	InsecureSkipTLS bool
	ProjectID       string
}

// SpeedyModeFake selects a local fake Speedy client that returns canned
// responses instead of calling the real Speedy Web API — used to exercise
// delivery methods, shipment creation and tracking in dev without a real
// carrier account or live parcels.
const SpeedyModeFake = "fake"

type FulfillmentConfig struct {
	// SpeedyMode is "real" (default) or "fake". Anything other than "fake"
	// keeps the real HTTP client, so production is never accidentally faked.
	SpeedyMode string
	// PollInterval controls how often the tracking poller runs. Kept short in
	// dev (paired with the fake client's time-based progression) so an order
	// visibly moves through statuses within minutes rather than hours.
	PollInterval time.Duration
}

const (
	// RevolutModeSandbox targets the Revolut Merchant sandbox environment
	// (test cards, no real money); RevolutModeProd targets live merchant.
	RevolutModeSandbox = "sandbox"
	RevolutModeProd    = "prod"
)

// PaymentsConfig carries the Revolut Merchant credentials and environment
// selector. Mode picks sandbox vs live endpoints; APIKey is the server-side
// Bearer secret; WebhookSecret verifies inbound webhook signatures. When
// APIKey is empty the checkout module falls back to the mock gateway, so
// local/devbox runs need no Revolut account.
type PaymentsConfig struct {
	RevolutMode          string
	RevolutAPIKey        string
	RevolutWebhookSecret string
	RevolutAPIVersion    string
}

// RevolutBaseURL returns the Merchant API base URL for the configured mode.
func (c PaymentsConfig) RevolutBaseURL() string {
	if c.RevolutMode == RevolutModeProd {
		return "https://merchant.revolut.com"
	}
	return "https://sandbox-merchant.revolut.com"
}

// LoadConfig loads configuration from environment variables.
func LoadConfig() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	sessionTTL := 30 * 24 * time.Hour
	if v := os.Getenv("AUTH_SESSION_TTL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("invalid AUTH_SESSION_TTL: %w", err)
		}
		sessionTTL = d
	}

	pollInterval := 15 * time.Minute
	if v := os.Getenv("FULFILLMENT_POLL_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("invalid FULFILLMENT_POLL_INTERVAL: %w", err)
		}
		pollInterval = d
	}

	cfg := &Config{
		App: AppConfig{
			Name: getEnv("APP_NAME", "fashion-store-api"),
			Env:  getEnv("APP_ENV", "local"),
		},
		HTTP: HTTPConfig{
			Addr: getEnv("HTTP_ADDR", ":8080"),
		},
		Database: DatabaseConfig{
			URL: dbURL,
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Google: GoogleConfig{
			ClientID: os.Getenv("GOOGLE_CLIENT_ID"),
		},
		Auth: AuthConfig{
			SigningSecret: os.Getenv("AUTH_SIGNING_SECRET"),
			SessionTTL:    sessionTTL,
		},
		Storage: StorageConfig{
			Endpoint:        getEnv("STORAGE_ENDPOINT", "https://fakegcs.shared.svc.cluster.local:4443"),
			Bucket:          getEnv("STORAGE_BUCKET", "product-media"),
			InsecureSkipTLS: getEnv("STORAGE_INSECURE_SKIP_TLS", "true") == "true",
			ProjectID:       os.Getenv("STORAGE_PROJECT_ID"),
		},
		Fulfillment: FulfillmentConfig{
			SpeedyMode:   getEnv("SPEEDY_MODE", "real"),
			PollInterval: pollInterval,
		},
		Payments: PaymentsConfig{
			RevolutMode:          getEnv("REVOLUT_MODE", RevolutModeSandbox),
			RevolutAPIKey:        os.Getenv("REVOLUT_API_KEY"),
			RevolutWebhookSecret: os.Getenv("REVOLUT_WEBHOOK_SECRET"),
			RevolutAPIVersion:    os.Getenv("REVOLUT_API_VERSION"),
		},
	}

	if cfg.Payments.RevolutMode != RevolutModeSandbox && cfg.Payments.RevolutMode != RevolutModeProd {
		return nil, fmt.Errorf("invalid REVOLUT_MODE %q: must be %q or %q", cfg.Payments.RevolutMode, RevolutModeSandbox, RevolutModeProd)
	}

	// Fail closed in production: never let a prod deploy come up pointed at the
	// Revolut sandbox, or without the credentials needed to charge and to
	// verify webhooks. Mirrors the fail-closed CORS stance for prod.
	if cfg.App.Env == RevolutModeProd {
		if cfg.Payments.RevolutMode != RevolutModeProd {
			return nil, fmt.Errorf("REVOLUT_MODE must be %q when APP_ENV=prod", RevolutModeProd)
		}
		if cfg.Payments.RevolutAPIKey == "" {
			return nil, fmt.Errorf("REVOLUT_API_KEY is required when APP_ENV=prod")
		}
		if cfg.Payments.RevolutWebhookSecret == "" {
			return nil, fmt.Errorf("REVOLUT_WEBHOOK_SECRET is required when APP_ENV=prod")
		}
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
