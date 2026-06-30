package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Module adapts Handler to the app.RouteRegistrar interface, binding the
// requireAuth middleware supplied by the auth module at wiring time.
type Module struct {
	handler     *Handler
	requireAuth func(http.Handler) http.Handler
}

func NewModule(handler *Handler, requireAuth func(http.Handler) http.Handler) *Module {
	return &Module{handler: handler, requireAuth: requireAuth}
}

func (m *Module) RegisterRoutes(r chi.Router) {
	m.handler.RegisterRoutes(r, m.requireAuth)
}
