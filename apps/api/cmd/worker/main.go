package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adotomov/fashion-store/apps/api/internal/app"
	"github.com/adotomov/fashion-store/apps/api/internal/platform/telemetry"
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

	obsCfg := bootstrapped.Config.Observability
	shutdownTelemetry, err := telemetry.Setup(ctx, telemetry.Config{
		ProjectID:      obsCfg.ProjectID,
		ServiceName:    bootstrapped.Config.App.Name + "-worker",
		Env:            bootstrapped.Config.App.Env,
		TracesEnabled:  obsCfg.TracesEnabled,
		MetricsEnabled: obsCfg.MetricsEnabled,
		SampleRatio:    obsCfg.SampleRatio,
		MetricInterval: obsCfg.MetricInterval,
	})
	if err != nil {
		log.Error("telemetry setup failed", slog.Any("error", err))
		os.Exit(1)
	}

	log.Info("worker started")

	<-ctx.Done()
	log.Info("worker shutting down")

	flushCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := shutdownTelemetry(flushCtx); err != nil {
		log.Error("telemetry shutdown failed", slog.Any("error", err))
	}
}
