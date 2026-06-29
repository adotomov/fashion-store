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

type ProductTypeHandler struct {
	service *application.ProductTypeService
}

func NewProductTypeHandler(service *application.ProductTypeService) *ProductTypeHandler {
	return &ProductTypeHandler{service: service}
}

func (h *ProductTypeHandler) RegisterRoutes(r chi.Router, requireAdmin func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(requireAdmin)
		r.Route("/admin/product-types", func(r chi.Router) {
			r.Get("/", h.list)
			r.Post("/", h.create)
			r.Patch("/{id}", h.update)
			r.Delete("/{id}", h.delete)
		})
	})
}

type productTypeRequest struct {
	Name string `json:"name"`
}

type productTypeResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Position  int    `json:"position"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toProductTypeResponse(t domain.ProductType) productTypeResponse {
	return productTypeResponse{
		ID:        t.ID.String(),
		Name:      t.Name,
		Slug:      t.Slug,
		Position:  t.Position,
		CreatedAt: t.CreatedAt.Format(timeFormat),
		UpdatedAt: t.UpdatedAt.Format(timeFormat),
	}
}

func (h *ProductTypeHandler) create(w http.ResponseWriter, r *http.Request) {
	var req productTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	productType, err := h.service.CreateProductType(r.Context(), application.CreateProductTypeInput{Name: req.Name})
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toProductTypeResponse(*productType))
}

func (h *ProductTypeHandler) list(w http.ResponseWriter, r *http.Request) {
	productTypes, err := h.service.ListProductTypes(r.Context())
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	resp := make([]productTypeResponse, 0, len(productTypes))
	for _, t := range productTypes {
		resp = append(resp, toProductTypeResponse(t))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *ProductTypeHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "product type id is invalid")
		return
	}

	var req struct {
		Name     *string `json:"name,omitempty"`
		Position *int    `json:"position,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	productType, err := h.service.UpdateProductType(r.Context(), id, application.UpdateProductTypeInput{
		Name:     req.Name,
		Position: req.Position,
	})
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toProductTypeResponse(*productType))
}

func (h *ProductTypeHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "product type id is invalid")
		return
	}

	if err := h.service.DeleteProductType(r.Context(), id); err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
