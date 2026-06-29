package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/adotomov/fashion-store/apps/api/internal/app"
)

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
	log.Info("worker started")

	<-ctx.Done()
	log.Info("worker shutting down")
}
