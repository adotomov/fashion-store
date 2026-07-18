package app

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// RouteRegistrar is implemented by domain modules to register their HTTP
// routes onto the versioned API router.
type RouteRegistrar interface {
	RegisterRoutes(r chi.Router)
}

// RootRouteRegistrar is optionally implemented by a module that also needs to
// mount routes at the router root, outside /api/v1 and its auth/CORS — e.g. a
// payment-provider webhook authenticated by signature rather than a token.
type RootRouteRegistrar interface {
	RegisterRootRoutes(r chi.Router)
}

// maxRequestBodyBytes bounds request body size to prevent memory-exhaustion
// DoS. Sits above the per-route media-upload cap (10 MiB) so uploads still work.
const maxRequestBodyBytes = 16 << 20 // 16 MiB

// readyzTimeout bounds the readiness DB ping so a hung database surfaces as a
// fast 503 rather than a stalled health check.
const readyzTimeout = 2 * time.Second

// NewRouter builds the application's HTTP router: health endpoints are
// unversioned, domain module routes are mounted under /api/v1. enableHSTS
// turns on the Strict-Transport-Security header (deployed HTTPS environments
// only). When obs.TracesEnabled is set the whole handler is wrapped in the
// otelhttp span middleware so each request becomes a Cloud Trace span.
func NewRouter(log *slog.Logger, pool *pgxpool.Pool, obs ObservabilityConfig, corsOrigins []string, enableHSTS bool, registrars ...RouteRegistrar) http.Handler {
	r := chi.NewRouter()

	// RequestID and LogContext seed the log-correlation attrs; Recover runs
	// inside them so panic logs carry request_id/trace, and RequestLogging wraps
	// Recover so it still records the final (500) status.
	r.Use(RequestID)
	r.Use(LogContext(obs.ProjectID))
	r.Use(RequestLogging(log))
	r.Use(Recover(log))
	r.Use(SecurityHeaders(enableHSTS))
	r.Use(MaxBodyBytes(maxRequestBodyBytes))
	r.Use(CORS(corsOrigins))

	r.Get("/healthz", healthzHandler)
	r.Get("/readyz", readyzHandler(pool))

	// Root-level routes (e.g. provider webhooks) mount before the versioned
	// API, outside its auth.
	for _, reg := range registrars {
		if rr, ok := reg.(RootRouteRegistrar); ok {
			rr.RegisterRootRoutes(r)
		}
	}

	r.Route("/api/v1", func(api chi.Router) {
		for _, reg := range registrars {
			reg.RegisterRoutes(api)
		}
	})

	if obs.TracesEnabled {
		// otelhttp is outermost so the server span covers all middleware. Spans
		// are named by method only to keep the name cardinality low (the chi
		// route pattern isn't matched yet at this outer layer); the full path is
		// captured as a span attribute by otelhttp.
		return otelhttp.NewHandler(r, "http.server",
			otelhttp.WithSpanNameFormatter(func(_ string, req *http.Request) string {
				return "HTTP " + req.Method
			}),
		)
	}
	return r
}

// healthzHandler is a liveness probe: it returns 200 as long as the process is
// serving, independent of downstream dependencies.
func healthzHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// readyzHandler is a readiness probe: it returns 503 when the database is
// unreachable so the load balancer and uptime checks stop routing to an
// instance that cannot serve requests.
func readyzHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readyzTimeout)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
