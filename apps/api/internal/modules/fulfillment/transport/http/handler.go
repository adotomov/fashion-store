package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/fulfillment/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/fulfillment/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
)

type Handler struct {
	service *application.Service
	// speedyDevMode is true when the API is wired to the fake Speedy client
	// (SPEEDY_MODE=fake). Surfaced to the admin UI so it's obvious the
	// integration is simulated, not talking to the real carrier.
	speedyDevMode bool
}

func NewHandler(service *application.Service, speedyDevMode bool) *Handler {
	return &Handler{service: service, speedyDevMode: speedyDevMode}
}

func (h *Handler) RegisterRoutes(r chi.Router, requireAdmin func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(requireAdmin)
		r.Get("/admin/logistics/providers", h.adminListProviders)
		r.Put("/admin/logistics/providers/{provider}", h.adminSaveProvider)
	})

	r.Get("/logistics/offices", h.searchOffices)
}

func (h *Handler) adminListProviders(w http.ResponseWriter, r *http.Request) {
	stored, err := h.service.ListSettings(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	byProvider := make(map[string]domain.ProviderSettings, len(stored))
	for _, s := range stored {
		byProvider[s.Provider] = s
	}

	resp := make([]providerResponse, 0, len(knownProviders))
	for _, p := range knownProviders {
		var pr providerResponse
		if settings, ok := byProvider[p.Code]; ok {
			pr = toProviderResponse(p.Code, p.Name, &settings)
		} else {
			pr = toProviderResponse(p.Code, p.Name, nil)
		}
		pr.DevMode = h.speedyDevMode && p.Code == domain.ProviderSpeedy
		resp = append(resp, pr)
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) adminSaveProvider(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")

	var req saveProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	// A masked placeholder means "unchanged" — strip it so the service's
	// merge logic preserves whatever is already stored.
	config := make(map[string]string, len(req.Config))
	for k, v := range req.Config {
		if maskedConfigKeys[k] && v == "********" {
			continue
		}
		config[k] = v
	}

	settings, err := h.service.SaveSettings(r.Context(), provider, req.Enabled, config)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	name := provider
	for _, p := range knownProviders {
		if p.Code == provider {
			name = p.Name
		}
	}
	resp := toProviderResponse(provider, name, settings)
	resp.DevMode = h.speedyDevMode && provider == domain.ProviderSpeedy
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) searchOffices(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	if provider == "" {
		provider = domain.ProviderSpeedy
	}
	officeType := r.URL.Query().Get("type")
	if officeType == "" {
		officeType = "APT"
	}
	city := r.URL.Query().Get("city")
	if city == "" {
		httpx.WriteError(w, http.StatusBadRequest, "missing_city", "city is required")
		return
	}

	offices, err := h.service.SearchOffices(r.Context(), provider, city, officeType)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toOfficeResponses(offices))
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrProviderDisabled):
		httpx.WriteError(w, http.StatusConflict, "provider_disabled", "logistics provider is disabled")
	case errors.Is(err, domain.ErrProviderNotFound):
		httpx.WriteError(w, http.StatusNotFound, "provider_not_found", "logistics provider not found")
	case errors.As(err, new(domain.ValidationError)):
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", err.Error())
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
