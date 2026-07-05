package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

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
