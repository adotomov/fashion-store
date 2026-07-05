package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

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
		r.Get("/admin/hero", h.getHeroSettings)
		r.Put("/admin/hero", h.saveHeroSettings)
		r.Post("/admin/hero/background", h.uploadHeroBackground)
		r.Delete("/admin/hero/background", h.deleteHeroBackground)
		r.Get("/admin/home-sections", h.listHomeSections)
		r.Put("/admin/home-sections/{sectionId}", h.saveHomeSection)
		r.Get("/admin/home-sections/{sectionId}/products", h.getSectionProducts)
		r.Put("/admin/home-sections/{sectionId}/products", h.setSectionProducts)
	})
	// Public routes — no admin auth required.
	r.Get("/storefront/hero", h.getHeroSettingsPublic)
	r.Get("/storefront/hero/background/file", h.serveHeroBackground)
	r.Get("/storefront/home-sections", h.listHomeSectionsPublic)
	r.Get("/storefront/home-sections/{sectionId}/product-ids", h.getSectionProductIDsPublic)
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

type heroSettingsResponse struct {
	Eyebrow              string  `json:"eyebrow"`
	Heading              string  `json:"heading"`
	Subtext              string  `json:"subtext"`
	CTAPrimaryLabel      string  `json:"cta_primary_label"`
	CTAPrimaryURL        string  `json:"cta_primary_url"`
	CTASecondaryLabel    *string `json:"cta_secondary_label,omitempty"`
	CTASecondaryURL      *string `json:"cta_secondary_url,omitempty"`
	BackgroundImageURL   *string `json:"background_image_url,omitempty"`
	UpdatedAt            string  `json:"updated_at"`
}

type saveHeroSettingsRequest struct {
	Eyebrow           string  `json:"eyebrow"`
	Heading           string  `json:"heading"`
	Subtext           string  `json:"subtext"`
	CTAPrimaryLabel   string  `json:"cta_primary_label"`
	CTAPrimaryURL     string  `json:"cta_primary_url"`
	CTASecondaryLabel *string `json:"cta_secondary_label"`
	CTASecondaryURL   *string `json:"cta_secondary_url"`
}

func toHeroSettingsResponse(s domain.HeroSettings) heroSettingsResponse {
	resp := heroSettingsResponse{
		Eyebrow:           s.Eyebrow,
		Heading:           s.Heading,
		Subtext:           s.Subtext,
		CTAPrimaryLabel:   s.CTAPrimaryLabel,
		CTAPrimaryURL:     s.CTAPrimaryURL,
		CTASecondaryLabel: s.CTASecondaryLabel,
		CTASecondaryURL:   s.CTASecondaryURL,
		UpdatedAt:         s.UpdatedAt.Format(timeFormat),
	}
	if s.HasBackground() {
		url := "/api/v1/storefront/hero/background/file"
		resp.BackgroundImageURL = &url
	}
	return resp
}

func (h *StoreSettingsHandler) getHeroSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.service.GetHeroSettings(r.Context())
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toHeroSettingsResponse(settings))
}

func (h *StoreSettingsHandler) getHeroSettingsPublic(w http.ResponseWriter, r *http.Request) {
	settings, err := h.service.GetHeroSettings(r.Context())
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toHeroSettingsResponse(settings))
}

func (h *StoreSettingsHandler) saveHeroSettings(w http.ResponseWriter, r *http.Request) {
	var req saveHeroSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	// Preserve existing background image when saving text fields only.
	current, err := h.service.GetHeroSettings(r.Context())
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	current.Eyebrow = req.Eyebrow
	current.Heading = req.Heading
	current.Subtext = req.Subtext
	current.CTAPrimaryLabel = req.CTAPrimaryLabel
	current.CTAPrimaryURL = req.CTAPrimaryURL
	current.CTASecondaryLabel = req.CTASecondaryLabel
	current.CTASecondaryURL = req.CTASecondaryURL

	settings, err := h.service.SaveHeroSettings(r.Context(), current)
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toHeroSettingsResponse(settings))
}

const maxHeroBackgroundUploadBytes = 20 << 20 // 20 MiB

func (h *StoreSettingsHandler) uploadHeroBackground(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxHeroBackgroundUploadBytes); err != nil {
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

	settings, err := h.service.UploadHeroBackground(r.Context(), header.Filename, contentType, file)
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toHeroSettingsResponse(settings))
}

func (h *StoreSettingsHandler) serveHeroBackground(w http.ResponseWriter, r *http.Request) {
	reader, contentType, err := h.service.OpenHeroBackground(r.Context())
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

func (h *StoreSettingsHandler) deleteHeroBackground(w http.ResponseWriter, r *http.Request) {
	settings, err := h.service.DeleteHeroBackground(r.Context())
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toHeroSettingsResponse(settings))
}

// ─── Home sections ────────────────────────────────────────────────────────────

type homeSectionResponse struct {
	ID        string `json:"id"`
	Enabled   bool   `json:"enabled"`
	Eyebrow   string `json:"eyebrow"`
	Heading   string `json:"heading"`
	UpdatedAt string `json:"updated_at"`
}

type saveHomeSectionRequest struct {
	Enabled bool   `json:"enabled"`
	Eyebrow string `json:"eyebrow"`
	Heading string `json:"heading"`
}

func toHomeSectionResponse(s domain.HomeSection) homeSectionResponse {
	return homeSectionResponse{
		ID:        s.ID,
		Enabled:   s.Enabled,
		Eyebrow:   s.Eyebrow,
		Heading:   s.Heading,
		UpdatedAt: s.UpdatedAt.Format(timeFormat),
	}
}

func (h *StoreSettingsHandler) listHomeSections(w http.ResponseWriter, r *http.Request) {
	sections, err := h.service.ListHomeSections(r.Context())
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	resp := make([]homeSectionResponse, len(sections))
	for i, s := range sections {
		resp[i] = toHomeSectionResponse(s)
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *StoreSettingsHandler) listHomeSectionsPublic(w http.ResponseWriter, r *http.Request) {
	sections, err := h.service.ListHomeSections(r.Context())
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	var resp []homeSectionResponse
	for _, s := range sections {
		if s.Enabled {
			resp = append(resp, toHomeSectionResponse(s))
		}
	}
	if resp == nil {
		resp = []homeSectionResponse{}
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *StoreSettingsHandler) saveHomeSection(w http.ResponseWriter, r *http.Request) {
	sectionID := chi.URLParam(r, "sectionId")
	var req saveHomeSectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}
	saved, err := h.service.SaveHomeSection(r.Context(), domain.HomeSection{
		ID:      sectionID,
		Enabled: req.Enabled,
		Eyebrow: req.Eyebrow,
		Heading: req.Heading,
	})
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toHomeSectionResponse(saved))
}

func (h *StoreSettingsHandler) getSectionProducts(w http.ResponseWriter, r *http.Request) {
	sectionID := chi.URLParam(r, "sectionId")
	ids, err := h.service.GetSectionProductIDs(r.Context(), sectionID)
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = id.String()
	}
	httpx.WriteJSON(w, http.StatusOK, strs)
}

func (h *StoreSettingsHandler) getSectionProductIDsPublic(w http.ResponseWriter, r *http.Request) {
	sectionID := chi.URLParam(r, "sectionId")
	ids, err := h.service.GetSectionProductIDs(r.Context(), sectionID)
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = id.String()
	}
	httpx.WriteJSON(w, http.StatusOK, strs)
}

func (h *StoreSettingsHandler) setSectionProducts(w http.ResponseWriter, r *http.Request) {
	sectionID := chi.URLParam(r, "sectionId")
	var rawIDs []string
	if err := json.NewDecoder(r.Body).Decode(&rawIDs); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "expected array of product ID strings")
		return
	}
	productIDs := make([]uuid.UUID, 0, len(rawIDs))
	for _, raw := range rawIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_product_id", "product ID is invalid: "+raw)
			return
		}
		productIDs = append(productIDs, id)
	}
	if err := h.service.SetSectionProducts(r.Context(), sectionID, productIDs); err != nil {
		writeAdminModuleError(w, err)
		return
	}
	strs := make([]string, len(productIDs))
	for i, id := range productIDs {
		strs[i] = id.String()
	}
	httpx.WriteJSON(w, http.StatusOK, strs)
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
	case errors.Is(err, domain.ErrHeroBackgroundNotFound):
		httpx.WriteError(w, http.StatusNotFound, "hero_background_not_found", "hero background image not found")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
