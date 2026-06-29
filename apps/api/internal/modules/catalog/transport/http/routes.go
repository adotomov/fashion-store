package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Module adapts the catalog module's handlers to the app.RouteRegistrar
// interface, binding the admin-role middleware supplied by the auth module
// at wiring time.
type Module struct {
	catalogHandler     *CatalogHandler
	categoryHandler    *CategoryHandler
	productTypeHandler *ProductTypeHandler
	attributeHandler   *AttributeHandler
	productHandler     *ProductHandler
	requireAdmin       func(http.Handler) http.Handler
}

func NewModule(
	catalogHandler *CatalogHandler,
	categoryHandler *CategoryHandler,
	productTypeHandler *ProductTypeHandler,
	attributeHandler *AttributeHandler,
	productHandler *ProductHandler,
	requireAdmin func(http.Handler) http.Handler,
) *Module {
	return &Module{
		catalogHandler:     catalogHandler,
		categoryHandler:    categoryHandler,
		productTypeHandler: productTypeHandler,
		attributeHandler:   attributeHandler,
		productHandler:     productHandler,
		requireAdmin:       requireAdmin,
	}
}

func (m *Module) RegisterRoutes(r chi.Router) {
	m.catalogHandler.RegisterRoutes(r, m.requireAdmin)
	m.categoryHandler.RegisterRoutes(r, m.requireAdmin)
	m.productTypeHandler.RegisterRoutes(r, m.requireAdmin)
	m.attributeHandler.RegisterRoutes(r, m.requireAdmin)
	m.productHandler.RegisterRoutes(r, m.requireAdmin)
}
