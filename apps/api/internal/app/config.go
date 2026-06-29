package app

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	App      AppConfig
	HTTP     HTTPConfig
	Database DatabaseConfig
	Log      LogConfig
	Google   GoogleConfig
	Auth     AuthConfig
	Storage  StorageConfig
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
		},
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
