package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/i18n/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/i18n/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
)

type LanguageHandler struct {
	service *application.LanguageService
}

func NewLanguageHandler(service *application.LanguageService) *LanguageHandler {
	return &LanguageHandler{service: service}
}

func (h *LanguageHandler) RegisterRoutes(r chi.Router, requireAdmin func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(requireAdmin)
		r.Route("/admin/languages", func(r chi.Router) {
			r.Get("/", h.list)
			r.Post("/", h.add)
			r.Patch("/{code}", h.setEnabled)
			r.Delete("/{code}", h.delete)
		})
	})
}

func (h *LanguageHandler) RegisterStorefrontRoutes(r chi.Router) {
	r.Get("/storefront/languages", h.listEnabled)
}

type languageResponse struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
	Enabled   bool   `json:"enabled"`
}

func toLanguageResponse(l domain.Language) languageResponse {
	return languageResponse{Code: l.Code, Name: l.Name, IsDefault: l.IsDefault, Enabled: l.Enabled}
}

func (h *LanguageHandler) list(w http.ResponseWriter, r *http.Request) {
	langs, err := h.service.List(r.Context())
	if err != nil {
		writeI18nError(w, err)
		return
	}
	resp := make([]languageResponse, 0, len(langs))
	for _, l := range langs {
		resp = append(resp, toLanguageResponse(l))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *LanguageHandler) listEnabled(w http.ResponseWriter, r *http.Request) {
	langs, err := h.service.ListEnabled(r.Context())
	if err != nil {
		writeI18nError(w, err)
		return
	}
	resp := make([]languageResponse, 0, len(langs))
	for _, l := range langs {
		resp = append(resp, toLanguageResponse(l))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

type addLanguageRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func (h *LanguageHandler) add(w http.ResponseWriter, r *http.Request) {
	var req addLanguageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}
	lang, err := h.service.Add(r.Context(), req.Code, req.Name)
	if err != nil {
		writeI18nError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toLanguageResponse(*lang))
}

type setEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

func (h *LanguageHandler) setEnabled(w http.ResponseWriter, r *http.Request) {
	var req setEnabledRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}
	lang, err := h.service.SetEnabled(r.Context(), chi.URLParam(r, "code"), req.Enabled)
	if err != nil {
		writeI18nError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toLanguageResponse(*lang))
}

func (h *LanguageHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), chi.URLParam(r, "code")); err != nil {
		writeI18nError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeI18nError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrLanguageNotFound):
		httpx.WriteError(w, http.StatusNotFound, "language_not_found", "language not found")
	case errors.Is(err, domain.ErrLanguageAlreadyExists):
		httpx.WriteError(w, http.StatusConflict, "language_already_exists", "this language code already exists")
	case errors.Is(err, domain.ErrCannotModifyDefaultLocale):
		httpx.WriteError(w, http.StatusBadRequest, "cannot_modify_default_locale", "the default language cannot be disabled or removed")
	case errors.Is(err, domain.ErrInvalidLanguageCode):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_language_code", "language code and name are required")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
