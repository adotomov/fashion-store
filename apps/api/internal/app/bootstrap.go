package app

import (
	"context"
	"log/slog"

	"github.com/adotomov/fashion-store/apps/api/internal/platform/logger"
	"github.com/adotomov/fashion-store/apps/api/internal/platform/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

// App holds the process-level dependencies shared by the API and worker
// binaries. It does not build the HTTP server, since the API binary needs
// to wire in domain module route registrars after Bootstrap returns.
type App struct {
	Config *Config
	Logger *slog.Logger
	DB     *pgxpool.Pool
}

// Bootstrap loads config, initializes logging, and connects to the
// database. Callers are responsible for closing App.DB when done.
func Bootstrap(ctx context.Context) (*App, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	log := logger.New(cfg.Log.Level, cfg.Log.Format, cfg.App.Name, cfg.App.Env)

	db, err := postgres.NewPool(ctx, cfg.Database.URL)
	if err != nil {
		return nil, err
	}

	return &App{
		Config: cfg,
		Logger: log,
		DB:     db,
	}, nil
}
