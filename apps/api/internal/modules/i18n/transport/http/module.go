package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Module adapts the i18n module's admin-gated handlers to the
// app.RouteRegistrar interface.
type Module struct {
	languageHandler    *LanguageHandler
	translationHandler *TranslationHandler
	uiStringHandler    *UIStringHandler
	requireAdmin       func(http.Handler) http.Handler
}

func NewModule(languageHandler *LanguageHandler, translationHandler *TranslationHandler, uiStringHandler *UIStringHandler, requireAdmin func(http.Handler) http.Handler) *Module {
	return &Module{
		languageHandler:    languageHandler,
		translationHandler: translationHandler,
		uiStringHandler:    uiStringHandler,
		requireAdmin:       requireAdmin,
	}
}

func (m *Module) RegisterRoutes(r chi.Router) {
	m.languageHandler.RegisterRoutes(r, m.requireAdmin)
	m.translationHandler.RegisterRoutes(r, m.requireAdmin)
	m.uiStringHandler.RegisterRoutes(r, m.requireAdmin)
}

// StorefrontModule exposes the public, unauthenticated read endpoints:
// enabled languages list and the ui-strings map for a given locale.
type StorefrontModule struct {
	languageHandler *LanguageHandler
	uiStringHandler *UIStringHandler
}

func NewStorefrontModule(languageHandler *LanguageHandler, uiStringHandler *UIStringHandler) *StorefrontModule {
	return &StorefrontModule{languageHandler: languageHandler, uiStringHandler: uiStringHandler}
}

func (m *StorefrontModule) RegisterRoutes(r chi.Router) {
	m.languageHandler.RegisterStorefrontRoutes(r)
	m.uiStringHandler.RegisterStorefrontRoutes(r)
}
