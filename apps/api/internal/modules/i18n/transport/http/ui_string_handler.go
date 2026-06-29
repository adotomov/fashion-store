package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/i18n/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/i18n/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
)

type UIStringHandler struct {
	service *application.UIStringService
}

func NewUIStringHandler(service *application.UIStringService) *UIStringHandler {
	return &UIStringHandler{service: service}
}

func (h *UIStringHandler) RegisterRoutes(r chi.Router, requireAdmin func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(requireAdmin)
		r.Route("/admin/ui-strings", func(r chi.Router) {
			r.Get("/", h.listAll)
			r.Put("/", h.upsert)
		})
	})
}

func (h *UIStringHandler) RegisterStorefrontRoutes(r chi.Router) {
	r.Get("/storefront/ui-strings", h.getByLocale)
}

type uiStringResponse struct {
	Key    string `json:"key"`
	Locale string `json:"locale"`
	Value  string `json:"value"`
}

func (h *UIStringHandler) listAll(w http.ResponseWriter, r *http.Request) {
	strings, err := h.service.ListAll(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
		return
	}
	resp := make([]uiStringResponse, 0, len(strings))
	for _, s := range strings {
		resp = append(resp, uiStringResponse{Key: s.Key, Locale: s.Locale, Value: s.Value})
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

type upsertUIStringRequest struct {
	Key    string `json:"key"`
	Locale string `json:"locale"`
	Value  string `json:"value"`
}

func (h *UIStringHandler) upsert(w http.ResponseWriter, r *http.Request) {
	var req upsertUIStringRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}
	if req.Key == "" || req.Locale == "" {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "key and locale are required")
		return
	}
	if err := h.service.Upsert(r.Context(), req.Key, req.Locale, req.Value); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *UIStringHandler) getByLocale(w http.ResponseWriter, r *http.Request) {
	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = domain.DefaultLocale
	}
	strings, err := h.service.GetByLocale(r.Context(), locale)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, strings)
}
