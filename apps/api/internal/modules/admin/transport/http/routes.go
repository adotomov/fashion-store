package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Module adapts the admin module's gated handlers to the app.RouteRegistrar
// interface, binding the admin-role middleware supplied by the auth module
// at wiring time.
type Module struct {
	storeSettingsHandler *StoreSettingsHandler
	storeAddressHandler  *StoreAddressHandler
	storeDocumentHandler *StoreDocumentHandler
	requireAdmin         func(http.Handler) http.Handler
}

func NewModule(storeSettingsHandler *StoreSettingsHandler, storeAddressHandler *StoreAddressHandler, storeDocumentHandler *StoreDocumentHandler, requireAdmin func(http.Handler) http.Handler) *Module {
	return &Module{
		storeSettingsHandler: storeSettingsHandler,
		storeAddressHandler:  storeAddressHandler,
		storeDocumentHandler: storeDocumentHandler,
		requireAdmin:         requireAdmin,
	}
}

func (m *Module) RegisterRoutes(r chi.Router) {
	m.storeSettingsHandler.RegisterRoutes(r, m.requireAdmin)
	m.storeAddressHandler.RegisterRoutes(r, m.requireAdmin)
	m.storeDocumentHandler.RegisterRoutes(r, m.requireAdmin)
}
