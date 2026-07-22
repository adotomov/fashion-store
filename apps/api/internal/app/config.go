package app

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	App           AppConfig
	HTTP          HTTPConfig
	Database      DatabaseConfig
	Log           LogConfig
	Google        GoogleConfig
	Auth          AuthConfig
	Storage       StorageConfig
	Fulfillment   FulfillmentConfig
	Payments      PaymentsConfig
	Email         EmailConfig
	Observability ObservabilityConfig
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

// ObservabilityConfig controls structured-log trace correlation and the OTel
// telemetry pipeline (Cloud Trace + Cloud Monitoring). Traces and metrics
// default OFF so local/devbox runs need no GCP credentials; Cloud Run enables
// them via env. ProjectID is required for trace correlation and metric export
// and is reused from the storage config's project when a dedicated
// GCP_PROJECT_ID is not set.
type ObservabilityConfig struct {
	ProjectID      string
	TracesEnabled  bool
	MetricsEnabled bool
	// SampleRatio is the parent-based trace sampling ratio (0.0–1.0) applied to
	// root spans; child spans follow the parent's decision.
	SampleRatio float64
	// MetricInterval is how often the OTel meter provider pushes to Cloud
	// Monitoring.
	MetricInterval time.Duration
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

const (
	// EmailModeLog renders emails and writes them to the log instead of
	// delivering them — the default, so local/devbox runs need no SendGrid
	// account and no real mail can escape during development.
	EmailModeLog = "log"
	// EmailModeSendGrid delivers through the SendGrid v3 API.
	EmailModeSendGrid = "sendgrid"
)

// EmailConfig carries the transactional email settings. Mode selects the sender
// implementation and SendGridAPIKey is the server-side Bearer secret. From is
// fixed per environment (info@verani.bg) rather than chosen per message, so no
// producer can send under an address the domain has not authenticated.
type EmailConfig struct {
	Mode           string
	SendGridAPIKey string
	FromAddress    string
	FromName       string
	// WebhookVerificationKey is SendGrid's base64 DER ECDSA *public* key for the
	// Signed Event Webhook (not a shared secret — SendGrid signs, we verify).
	// Empty means inbound events are rejected, so the endpoint fails closed.
	WebhookVerificationKey string
	DispatchInterval       time.Duration
	// StorefrontURL is the public site base URL, linked from email bodies.
	StorefrontURL string
	// AssetBaseURL is the API's public base URL, used to turn the store logo's
	// relative path into an absolute one a mail client can load. When empty the
	// layout falls back to rendering the store name as text.
	AssetBaseURL string
}

// Enabled reports whether real delivery is configured. Anything else falls back
// to the log sender, mirroring how an absent REVOLUT_API_KEY selects the mock
// payment gateway.
func (c EmailConfig) Enabled() bool {
	return c.Mode == EmailModeSendGrid && c.SendGridAPIKey != ""
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

	sampleRatio := 0.1
	if v := os.Getenv("OTEL_TRACE_SAMPLE_RATIO"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid OTEL_TRACE_SAMPLE_RATIO: %w", err)
		}
		sampleRatio = f
	}

	metricInterval := 60 * time.Second
	if v := os.Getenv("OTEL_METRIC_EXPORT_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("invalid OTEL_METRIC_EXPORT_INTERVAL: %w", err)
		}
		metricInterval = d
	}

	emailDispatchInterval := 15 * time.Second
	if v := os.Getenv("EMAIL_DISPATCH_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("invalid EMAIL_DISPATCH_INTERVAL: %w", err)
		}
		emailDispatchInterval = d
	}

	// Reuse the storage project when a dedicated GCP_PROJECT_ID is unset — both
	// point at the same GCP project in every deployed environment.
	projectID := getEnv("GCP_PROJECT_ID", os.Getenv("STORAGE_PROJECT_ID"))

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
		Email: EmailConfig{
			Mode:             getEnv("EMAIL_MODE", EmailModeLog),
			SendGridAPIKey:   os.Getenv("SENDGRID_API_KEY"),
			FromAddress:      getEnv("EMAIL_FROM", "info@verani.bg"),
			FromName:         getEnv("EMAIL_FROM_NAME", "Verani"),
			WebhookVerificationKey: os.Getenv("EMAIL_WEBHOOK_VERIFICATION_KEY"),
			DispatchInterval:       emailDispatchInterval,
			StorefrontURL:    getEnv("STOREFRONT_URL", "http://localhost:5173"),
			AssetBaseURL:     os.Getenv("PUBLIC_API_URL"),
		},
		Observability: ObservabilityConfig{
			ProjectID:      projectID,
			TracesEnabled:  getEnv("OTEL_TRACES_ENABLED", "false") == "true",
			MetricsEnabled: getEnv("OTEL_METRICS_ENABLED", "false") == "true",
			SampleRatio:    sampleRatio,
			MetricInterval: metricInterval,
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

	// Catch a half-configured email setup at boot rather than dead-lettering
	// every message later. Email intentionally stays optional in every
	// environment (an env with EMAIL_MODE=log simply never delivers), but asking
	// for SendGrid without a key is always a mistake.
	if cfg.Email.Mode != EmailModeLog && cfg.Email.Mode != EmailModeSendGrid {
		return nil, fmt.Errorf("invalid EMAIL_MODE %q: must be %q or %q", cfg.Email.Mode, EmailModeLog, EmailModeSendGrid)
	}
	if cfg.Email.Mode == EmailModeSendGrid {
		if cfg.Email.SendGridAPIKey == "" {
			return nil, fmt.Errorf("SENDGRID_API_KEY is required when EMAIL_MODE=%q", EmailModeSendGrid)
		}
		if cfg.Email.FromAddress == "" {
			return nil, fmt.Errorf("EMAIL_FROM is required when EMAIL_MODE=%q", EmailModeSendGrid)
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
