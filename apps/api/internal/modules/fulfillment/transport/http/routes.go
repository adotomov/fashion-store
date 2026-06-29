package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Module adapts Handler to the app.RouteRegistrar interface, binding the
// admin-only middleware at wiring time. The offices lookup stays
// unauthenticated since checkout needs it for guest customers too.
type Module struct {
	handler      *Handler
	requireAdmin func(http.Handler) http.Handler
}

func NewModule(handler *Handler, requireAdmin func(http.Handler) http.Handler) *Module {
	return &Module{handler: handler, requireAdmin: requireAdmin}
}

func (m *Module) RegisterRoutes(r chi.Router) {
	m.handler.RegisterRoutes(r, m.requireAdmin)
}
