package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Module adapts Handler to the app.RouteRegistrar interface, binding the
// auth middlewares supplied by the auth module at wiring time.
type Module struct {
	handler      *Handler
	requireAuth  func(http.Handler) http.Handler
	requireAdmin func(http.Handler) http.Handler
}

func NewModule(handler *Handler, requireAuth, requireAdmin func(http.Handler) http.Handler) *Module {
	return &Module{handler: handler, requireAuth: requireAuth, requireAdmin: requireAdmin}
}

func (m *Module) RegisterRoutes(r chi.Router) {
	m.handler.RegisterRoutes(r, m.requireAuth, m.requireAdmin)
}
