package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/adotomov/fashion-store/apps/api/internal/app"
)

const shutdownTimeout = 15 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	bootstrapped, err := app.Bootstrap(ctx)
	if err != nil {
		slog.Error("bootstrap failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer bootstrapped.DB.Close()

	log := bootstrapped.Logger

	registrars, fulfillmentService := buildRegistrars(bootstrapped)

	router := app.NewRouter(log, corsOrigins(), registrars...)
	srv := app.NewServer(bootstrapped.Config.HTTP.Addr, router)

	go fulfillmentService.Run(ctx, 15*time.Minute)

	go func() {
		log.Info("starting api server", slog.String("addr", bootstrapped.Config.HTTP.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")

	if err := app.Shutdown(context.Background(), srv, shutdownTimeout); err != nil {
		log.Error("graceful shutdown failed", slog.Any("error", err))
	}
}

func corsOrigins() []string {
	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		return strings.Split(v, ",")
	}
	return []string{"http://localhost:5173"}
}
