package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/admin/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/admin/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
)

const timeFormat = "2006-01-02T15:04:05Z07:00"

type StoreSettingsHandler struct {
	service *application.StoreSettingsService
}

func NewStoreSettingsHandler(service *application.StoreSettingsService) *StoreSettingsHandler {
	return &StoreSettingsHandler{service: service}
}

func (h *StoreSettingsHandler) RegisterRoutes(r chi.Router, requireAdmin func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(requireAdmin)
		r.Route("/admin/store-settings", func(r chi.Router) {
			r.Get("/", h.get)
			r.Patch("/", h.update)
			r.Post("/logo", h.uploadLogo)
			r.Get("/logo/file", h.serveLogo)
			r.Delete("/logo", h.deleteLogo)
		})
	})
}

type storeSettingsResponse struct {
	StoreName          string  `json:"store_name"`
	LegalEntityName    *string `json:"legal_entity_name,omitempty"`
	Locale             string  `json:"locale"`
	Currency           string  `json:"currency"`
	ContactEmail       *string `json:"contact_email,omitempty"`
	ContactPhone       *string `json:"contact_phone,omitempty"`
	CompanyDescription *string `json:"company_description,omitempty"`
	LogoURL            *string `json:"logo_url,omitempty"`
	UpdatedAt          string  `json:"updated_at"`
}

// toStoreSettingsResponse is shared by both the admin handler and the public
// storefront handler — the data isn't sensitive, only the basePath (which
// proxy endpoint serves the logo) differs between the two.
func toStoreSettingsResponse(s domain.StoreSettings, basePath string) storeSettingsResponse {
	resp := storeSettingsResponse{
		StoreName:          s.StoreName,
		LegalEntityName:    s.LegalEntityName,
		Locale:             s.Locale,
		Currency:           s.Currency,
		ContactEmail:       s.ContactEmail,
		ContactPhone:       s.ContactPhone,
		CompanyDescription: s.CompanyDescription,
		UpdatedAt:          s.UpdatedAt.Format(timeFormat),
	}
	if s.HasLogo() {
		url := basePath + "/logo/file"
		resp.LogoURL = &url
	}
	return resp
}

func (h *StoreSettingsHandler) get(w http.ResponseWriter, r *http.Request) {
	settings, err := h.service.GetSettings(r.Context())
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toStoreSettingsResponse(*settings, "/api/v1/admin/store-settings"))
}

type updateStoreSettingsRequest struct {
	StoreName          *string `json:"store_name,omitempty"`
	LegalEntityName    *string `json:"legal_entity_name,omitempty"`
	Locale             *string `json:"locale,omitempty"`
	Currency           *string `json:"currency,omitempty"`
	ContactEmail       *string `json:"contact_email,omitempty"`
	ContactPhone       *string `json:"contact_phone,omitempty"`
	CompanyDescription *string `json:"company_description,omitempty"`
}

func (h *StoreSettingsHandler) update(w http.ResponseWriter, r *http.Request) {
	var req updateStoreSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	settings, err := h.service.UpdateSettings(r.Context(), application.UpdateStoreSettingsInput{
		StoreName:          req.StoreName,
		LegalEntityName:    req.LegalEntityName,
		Locale:             req.Locale,
		Currency:           req.Currency,
		ContactEmail:       req.ContactEmail,
		ContactPhone:       req.ContactPhone,
		CompanyDescription: req.CompanyDescription,
	})
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toStoreSettingsResponse(*settings, "/api/v1/admin/store-settings"))
}

const maxLogoUploadBytes = 10 << 20 // 10 MiB

func (h *StoreSettingsHandler) uploadLogo(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxLogoUploadBytes); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "could not parse multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "missing_file", "file is required")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	settings, err := h.service.UploadLogo(r.Context(), header.Filename, contentType, file)
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toStoreSettingsResponse(*settings, "/api/v1/admin/store-settings"))
}

// serveLogo proxies the stored object's bytes back to the client rather
// than pointing <img src> at the storage backend directly — FakeGCS's
// self-signed local cert would otherwise make the browser refuse to load it.
func (h *StoreSettingsHandler) serveLogo(w http.ResponseWriter, r *http.Request) {
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

func (h *StoreSettingsHandler) deleteLogo(w http.ResponseWriter, r *http.Request) {
	settings, err := h.service.DeleteLogo(r.Context())
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toStoreSettingsResponse(*settings, "/api/v1/admin/store-settings"))
}

func writeAdminModuleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrLogoNotFound):
		httpx.WriteError(w, http.StatusNotFound, "logo_not_found", "store logo not found")
	case errors.Is(err, domain.ErrDocumentNotFound):
		httpx.WriteError(w, http.StatusNotFound, "document_not_found", "document not found")
	case errors.Is(err, domain.ErrInvalidDocumentType):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_document_type", "document type must be 'terms' or 'privacy'")
	case errors.Is(err, domain.ErrAddressNotFound):
		httpx.WriteError(w, http.StatusNotFound, "address_not_found", "store address not found")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
