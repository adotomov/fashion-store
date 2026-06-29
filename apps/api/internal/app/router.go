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

// NewRouter builds the application's HTTP router: health endpoints are
// unversioned, domain module routes are mounted under /api/v1.
func NewRouter(log *slog.Logger, corsOrigins []string, registrars ...RouteRegistrar) http.Handler {
	r := chi.NewRouter()

	r.Use(Recover(log))
	r.Use(RequestID)
	r.Use(RequestLogging(log))
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
