package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
)

type AttributeHandler struct {
	service *application.AttributeService
}

func NewAttributeHandler(service *application.AttributeService) *AttributeHandler {
	return &AttributeHandler{service: service}
}

func (h *AttributeHandler) RegisterRoutes(r chi.Router, requireAdmin func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(requireAdmin)
		r.Route("/admin/attributes", func(r chi.Router) {
			r.Get("/", h.list)
			r.Post("/", h.create)
			r.Patch("/{id}", h.update)
			r.Delete("/{id}", h.delete)
			r.Post("/{id}/values", h.addValue)
			r.Delete("/{id}/values/{valueId}", h.deleteValue)
		})
	})
}

type attributeValueResponse struct {
	ID          string `json:"id"`
	AttributeID string `json:"attribute_id"`
	Value       string `json:"value"`
}

type attributeResponse struct {
	ID        string                   `json:"id"`
	Name      string                   `json:"name"`
	Values    []attributeValueResponse `json:"values"`
	CreatedAt string                   `json:"created_at"`
	UpdatedAt string                   `json:"updated_at"`
}

func toAttributeResponse(a domain.Attribute) attributeResponse {
	values := make([]attributeValueResponse, 0, len(a.Values))
	for _, v := range a.Values {
		values = append(values, attributeValueResponse{ID: v.ID.String(), AttributeID: v.AttributeID.String(), Value: v.Value})
	}
	return attributeResponse{
		ID:        a.ID.String(),
		Name:      a.Name,
		Values:    values,
		CreatedAt: a.CreatedAt.Format(timeFormat),
		UpdatedAt: a.UpdatedAt.Format(timeFormat),
	}
}

func (h *AttributeHandler) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	attribute, err := h.service.CreateAttribute(r.Context(), application.CreateAttributeInput{Name: req.Name})
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toAttributeResponse(*attribute))
}

func (h *AttributeHandler) list(w http.ResponseWriter, r *http.Request) {
	attributes, err := h.service.ListAttributes(r.Context())
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	resp := make([]attributeResponse, 0, len(attributes))
	for _, a := range attributes {
		resp = append(resp, toAttributeResponse(a))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *AttributeHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "attribute id is invalid")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	attribute, err := h.service.UpdateAttribute(r.Context(), id, req.Name)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toAttributeResponse(*attribute))
}

func (h *AttributeHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "attribute id is invalid")
		return
	}

	if err := h.service.DeleteAttribute(r.Context(), id); err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AttributeHandler) addValue(w http.ResponseWriter, r *http.Request) {
	attributeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "attribute id is invalid")
		return
	}

	var req struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	value, err := h.service.AddValue(r.Context(), attributeID, req.Value)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, attributeValueResponse{ID: value.ID.String(), AttributeID: value.AttributeID.String(), Value: value.Value})
}

func (h *AttributeHandler) deleteValue(w http.ResponseWriter, r *http.Request) {
	attributeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "attribute id is invalid")
		return
	}
	valueID, err := uuid.Parse(chi.URLParam(r, "valueId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "attribute value id is invalid")
		return
	}

	if err := h.service.DeleteValue(r.Context(), attributeID, valueID); err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
