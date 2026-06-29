package http

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
)

type CatalogHandler struct {
	service *application.CatalogService
}

func NewCatalogHandler(service *application.CatalogService) *CatalogHandler {
	return &CatalogHandler{service: service}
}

func (h *CatalogHandler) RegisterRoutes(r chi.Router, requireAdmin func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(requireAdmin)
		r.Route("/admin/catalogs", func(r chi.Router) {
			r.Get("/", h.list)
			r.Post("/", h.create)
			r.Get("/{id}", h.get)
			r.Patch("/{id}", h.update)
			r.Delete("/{id}", h.delete)
			r.Get("/{id}/export", h.export)
		})
	})
}

type createCatalogRequest struct {
	Name string `json:"name"`
}

func (h *CatalogHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createCatalogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	catalog, err := h.service.CreateCatalog(r.Context(), application.CreateCatalogInput{Name: req.Name})
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toCatalogResponse(*catalog))
}

func (h *CatalogHandler) list(w http.ResponseWriter, r *http.Request) {
	catalogs, err := h.service.ListCatalogs(r.Context())
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	resp := make([]catalogResponse, 0, len(catalogs))
	for _, c := range catalogs {
		resp = append(resp, toCatalogResponse(c))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *CatalogHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "catalog id is invalid")
		return
	}

	catalog, err := h.service.GetCatalog(r.Context(), id)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toCatalogResponse(*catalog))
}

type updateCatalogRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
}

func (h *CatalogHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "catalog id is invalid")
		return
	}

	var req updateCatalogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	input := application.UpdateCatalogInput{
		Name:        req.Name,
		Description: req.Description,
	}
	if req.Status != nil {
		status := domain.Status(*req.Status)
		input.Status = &status
	}

	catalog, err := h.service.UpdateCatalog(r.Context(), id, input)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toCatalogResponse(*catalog))
}

func (h *CatalogHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "catalog id is invalid")
		return
	}

	if err := h.service.DeleteCatalog(r.Context(), id); err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CatalogHandler) export(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "catalog id is invalid")
		return
	}

	catalog, err := h.service.GetCatalog(r.Context(), id)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	format := r.URL.Query().Get("format")
	switch format {
	case "json":
		filename := fmt.Sprintf("catalog-%s.json", catalog.Slug)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		httpx.WriteJSON(w, http.StatusOK, toCatalogResponse(*catalog))
	case "csv", "":
		filename := fmt.Sprintf("catalog-%s.csv", catalog.Slug)
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

		writer := csv.NewWriter(w)
		_ = writer.Write([]string{"id", "name", "slug", "description", "status", "created_at", "updated_at"})
		_ = writer.Write([]string{
			catalog.ID.String(),
			catalog.Name,
			catalog.Slug,
			catalog.Description,
			string(catalog.Status),
			catalog.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			catalog.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
		writer.Flush()
	default:
		httpx.WriteError(w, http.StatusBadRequest, "invalid_format", "format must be csv or json")
	}
}
