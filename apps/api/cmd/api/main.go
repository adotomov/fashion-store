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

	registrars, fulfillmentService, checkoutService := buildRegistrars(bootstrapped)

	// Emit HSTS everywhere except local dev (served over plain HTTP).
	enableHSTS := bootstrapped.Config.App.Env != "local"
	router := app.NewRouter(log, corsOrigins(), enableHSTS, registrars...)
	srv := app.NewServer(bootstrapped.Config.HTTP.Addr, router)

	go fulfillmentService.Run(ctx, bootstrapped.Config.Fulfillment.PollInterval)

	// Safety net for missed/lost payment webhooks: reconcile abandoned
	// pending_payment card orders on an interval.
	go checkoutService.RunPaymentSweeper(ctx, checkoutapplication.DefaultPaymentSweepInterval, checkoutapplication.DefaultAbandonedPaymentTTL)

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
