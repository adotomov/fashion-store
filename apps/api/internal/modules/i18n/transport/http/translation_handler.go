package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/i18n/application"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
)

// TranslationHandler exposes generic admin CRUD over the translations table,
// keyed by entity_type/entity_id — used by every catalog admin form (and any
// future entity) to set per-field, per-locale overrides without each module
// needing its own translation endpoints.
type TranslationHandler struct {
	service *application.TranslationService
}

func NewTranslationHandler(service *application.TranslationService) *TranslationHandler {
	return &TranslationHandler{service: service}
}

func (h *TranslationHandler) RegisterRoutes(r chi.Router, requireAdmin func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(requireAdmin)
		r.Route("/admin/translations/{entityType}/{entityId}", func(r chi.Router) {
			r.Get("/{locale}", h.get)
			r.Put("/{locale}", h.set)
		})
	})
}

func (h *TranslationHandler) get(w http.ResponseWriter, r *http.Request) {
	entityID, err := uuid.Parse(chi.URLParam(r, "entityId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_entity_id", "entity id must be a UUID")
		return
	}
	fields, err := h.service.Get(r.Context(), chi.URLParam(r, "entityType"), entityID, chi.URLParam(r, "locale"))
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, fields)
}

func (h *TranslationHandler) set(w http.ResponseWriter, r *http.Request) {
	entityID, err := uuid.Parse(chi.URLParam(r, "entityId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_entity_id", "entity id must be a UUID")
		return
	}

	var fields map[string]string
	if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	entityType := chi.URLParam(r, "entityType")
	locale := chi.URLParam(r, "locale")
	for field, value := range fields {
		if err := h.service.Set(r.Context(), entityType, entityID, locale, field, value); err != nil {
			writeI18nError(w, err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}
