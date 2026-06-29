package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Module adapts Handler to the app.RouteRegistrar interface, binding the
// optional/required auth middleware supplied by the auth module at wiring
// time.
type Module struct {
	handler      *Handler
	optionalAuth func(http.Handler) http.Handler
	requireAuth  func(http.Handler) http.Handler
}

func NewModule(handler *Handler, optionalAuth, requireAuth func(http.Handler) http.Handler) *Module {
	return &Module{handler: handler, optionalAuth: optionalAuth, requireAuth: requireAuth}
}

func (m *Module) RegisterRoutes(r chi.Router) {
	m.handler.RegisterRoutes(r, m.optionalAuth, m.requireAuth)
}
