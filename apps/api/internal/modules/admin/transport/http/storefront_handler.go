package http

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/admin/application"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
)

// StorefrontHandler exposes read-only, public (no admin auth) endpoints
// backing the customer-facing site: store identity/contact info for the
// footer and about page, plus file serving for the logo — mirrors
// catalog's StorefrontHandler.
type StorefrontHandler struct {
	service         *application.StoreSettingsService
	addressHandler  *StoreAddressHandler
	documentHandler *StoreDocumentHandler
}

func NewStorefrontHandler(service *application.StoreSettingsService, addressHandler *StoreAddressHandler, documentHandler *StoreDocumentHandler) *StorefrontHandler {
	return &StorefrontHandler{service: service, addressHandler: addressHandler, documentHandler: documentHandler}
}

func (h *StorefrontHandler) RegisterRoutes(r chi.Router) {
	r.Route("/storefront/store-settings", func(r chi.Router) {
		r.Get("/", h.get)
		r.Get("/logo/file", h.serveLogo)
	})
	h.addressHandler.RegisterStorefrontRoutes(r)
	h.documentHandler.RegisterStorefrontRoutes(r)
}

func (h *StorefrontHandler) get(w http.ResponseWriter, r *http.Request) {
	settings, err := h.service.GetSettings(r.Context())
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toStoreSettingsResponse(*settings, "/api/v1/storefront/store-settings"))
}

func (h *StorefrontHandler) serveLogo(w http.ResponseWriter, r *http.Request) {
	reader, contentType, err := h.service.OpenLogo(r.Context())
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	defer reader.Close()

	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	_, _ = io.Copy(w, reader)
}
