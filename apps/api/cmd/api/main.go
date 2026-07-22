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

	// Embeds the IANA timezone database into the binary so time.LoadLocation
	// (e.g. "Europe/Sofia" for invoice timestamps) works in the minimal Alpine
	// container image, which ships without tzdata.
	_ "time/tzdata"

	"github.com/adotomov/fashion-store/apps/api/internal/app"
	checkoutapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/checkout/application"
	"github.com/adotomov/fashion-store/apps/api/internal/platform/telemetry"
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

	// Telemetry (Cloud Trace + Cloud Monitoring) is opt-in via env; when off,
	// Setup and the returned shutdown are no-ops.
	obsCfg := bootstrapped.Config.Observability
	shutdownTelemetry, err := telemetry.Setup(ctx, telemetry.Config{
		ProjectID:      obsCfg.ProjectID,
		ServiceName:    bootstrapped.Config.App.Name,
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

	registrars, fulfillmentService, checkoutService, notificationsService := buildRegistrars(bootstrapped)

	// Emit HSTS everywhere except local dev (served over plain HTTP).
	enableHSTS := bootstrapped.Config.App.Env != "local"
	router := app.NewRouter(log, bootstrapped.DB, bootstrapped.Config.Observability, corsOrigins(), enableHSTS, registrars...)
	srv := app.NewServer(bootstrapped.Config.HTTP.Addr, router)

	go fulfillmentService.Run(ctx, bootstrapped.Config.Fulfillment.PollInterval)

	// Safety net for missed/lost payment webhooks: reconcile abandoned
	// pending_payment card orders on an interval.
	go checkoutService.RunPaymentSweeper(ctx, checkoutapplication.DefaultPaymentSweepInterval, checkoutapplication.DefaultAbandonedPaymentTTL)

	// Drains the transactional email outbox: producers only enqueue, so nothing
	// is actually delivered until this runs.
	go notificationsService.RunDispatcher(ctx, bootstrapped.Config.Email.DispatchInterval)

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

	// Flush buffered spans and the final metric batch before exiting.
	flushCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := shutdownTelemetry(flushCtx); err != nil {
		log.Error("telemetry shutdown failed", slog.Any("error", err))
	}
}

func corsOrigins() []string {
	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		return strings.Split(v, ",")
	}
	return []string{"http://localhost:5173"}
}
