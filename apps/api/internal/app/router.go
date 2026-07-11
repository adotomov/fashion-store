package app

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RouteRegistrar is implemented by domain modules to register their HTTP
// routes onto the versioned API router.
type RouteRegistrar interface {
	RegisterRoutes(r chi.Router)
}

// maxRequestBodyBytes bounds request body size to prevent memory-exhaustion
// DoS. Sits above the per-route media-upload cap (10 MiB) so uploads still work.
const maxRequestBodyBytes = 16 << 20 // 16 MiB

// NewRouter builds the application's HTTP router: health endpoints are
// unversioned, domain module routes are mounted under /api/v1. enableHSTS
// turns on the Strict-Transport-Security header (deployed HTTPS environments
// only).
func NewRouter(log *slog.Logger, corsOrigins []string, enableHSTS bool, registrars ...RouteRegistrar) http.Handler {
	r := chi.NewRouter()

	r.Use(Recover(log))
	r.Use(RequestID)
	r.Use(RequestLogging(log))
	r.Use(SecurityHeaders(enableHSTS))
	r.Use(MaxBodyBytes(maxRequestBodyBytes))
	r.Use(CORS(corsOrigins))

	r.Get("/healthz", healthzHandler)
	r.Get("/readyz", readyzHandler)

	r.Route("/api/v1", func(api chi.Router) {
		for _, reg := range registrars {
			reg.RegisterRoutes(api)
		}
	})

	return r
}

func healthzHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func readyzHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
