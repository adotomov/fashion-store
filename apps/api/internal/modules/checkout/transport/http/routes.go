package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Module adapts Handler to the app router. It binds the optional-auth
// middleware (checkout works for both signed-in and guest customers) and the
// admin middleware (for refunds) supplied by the auth module at wiring time.
type Module struct {
	handler      *Handler
	optionalAuth func(http.Handler) http.Handler
	requireAdmin func(http.Handler) http.Handler
}

func NewModule(handler *Handler, optionalAuth, requireAdmin func(http.Handler) http.Handler) *Module {
	return &Module{handler: handler, optionalAuth: optionalAuth, requireAdmin: requireAdmin}
}

func (m *Module) RegisterRoutes(r chi.Router) {
	m.handler.RegisterRoutes(r, m.optionalAuth, m.requireAdmin)
}

// RegisterRootRoutes mounts the Revolut webhook outside the /api/v1 tree and
// its auth — it's authenticated by the signature check, not a bearer token.
func (m *Module) RegisterRootRoutes(r chi.Router) {
	r.Post("/webhooks/revolut", m.handler.revolutWebhook)
}
