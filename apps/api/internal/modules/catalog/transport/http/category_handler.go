package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
)

type CategoryHandler struct {
	service *application.CategoryService
}

func NewCategoryHandler(service *application.CategoryService) *CategoryHandler {
	return &CategoryHandler{service: service}
}

func (h *CategoryHandler) RegisterRoutes(r chi.Router, requireAdmin func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(requireAdmin)
		r.Route("/admin/categories", func(r chi.Router) {
			r.Get("/", h.list)
			r.Post("/", h.create)
			r.Patch("/{id}", h.update)
			r.Delete("/{id}", h.delete)
			r.Post("/{id}/thumbnail", h.uploadThumbnail)
			r.Get("/{id}/thumbnail/file", h.serveThumbnail)
			r.Delete("/{id}/thumbnail", h.deleteThumbnail)
		})
	})
}

type categoryRequest struct {
	Name          string  `json:"name"`
	ParentID      *string `json:"parent_id,omitempty"`
	ProductTypeID string  `json:"product_type_id"`
}

type categoryResponse struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Slug          string  `json:"slug"`
	ParentID      *string `json:"parent_id,omitempty"`
	ProductTypeID string  `json:"product_type_id"`
	ImageURL      *string `json:"image_url,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

func toCategoryResponse(c domain.Category) categoryResponse {
	var parentID *string
	if c.ParentID != nil {
		s := c.ParentID.String()
		parentID = &s
	}
	resp := categoryResponse{
		ID:            c.ID.String(),
		Name:          c.Name,
		Slug:          c.Slug,
		ParentID:      parentID,
		ProductTypeID: c.ProductTypeID.String(),
		CreatedAt:     c.CreatedAt.Format(timeFormat),
		UpdatedAt:     c.UpdatedAt.Format(timeFormat),
	}
	if c.HasThumbnail() {
		url := "/api/v1/admin/categories/" + c.ID.String() + "/thumbnail/file"
		resp.ImageURL = &url
	}
	return resp
}

func parseParentID(raw *string) (*uuid.UUID, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(*raw)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (h *CategoryHandler) create(w http.ResponseWriter, r *http.Request) {
	var req categoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	parentID, err := parseParentID(req.ParentID)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_parent_id", "parent_id is invalid")
		return
	}

	productTypeID, err := uuid.Parse(req.ProductTypeID)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_product_type_id", "product_type_id is invalid")
		return
	}

	category, err := h.service.CreateCategory(r.Context(), application.CreateCategoryInput{
		Name:          req.Name,
		ParentID:      parentID,
		ProductTypeID: productTypeID,
	})
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toCategoryResponse(*category))
}

func (h *CategoryHandler) list(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.ListCategories(r.Context())
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	resp := make([]categoryResponse, 0, len(categories))
	for _, c := range categories {
		resp = append(resp, toCategoryResponse(c))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *CategoryHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "category id is invalid")
		return
	}

	var req struct {
		Name          *string `json:"name,omitempty"`
		ParentID      *string `json:"parent_id,omitempty"`
		ProductTypeID *string `json:"product_type_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	parentID, err := parseParentID(req.ParentID)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_parent_id", "parent_id is invalid")
		return
	}

	input := application.UpdateCategoryInput{Name: req.Name}
	if req.ParentID != nil {
		input.ParentID = parentID
	}
	if req.ProductTypeID != nil {
		productTypeID, err := uuid.Parse(*req.ProductTypeID)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_product_type_id", "product_type_id is invalid")
			return
		}
		input.ProductTypeID = &productTypeID
	}

	category, err := h.service.UpdateCategory(r.Context(), id, input)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toCategoryResponse(*category))
}

func (h *CategoryHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "category id is invalid")
		return
	}

	if err := h.service.DeleteCategory(r.Context(), id); err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CategoryHandler) uploadThumbnail(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "category id is invalid")
		return
	}

	const maxUploadBytes = 10 << 20 // 10 MiB
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
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

	category, err := h.service.UploadThumbnail(r.Context(), id, header.Filename, contentType, file)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toCategoryResponse(*category))
}

// serveThumbnail proxies the stored object's bytes back to the client. The
// admin UI's <img src> points here rather than at the storage backend
// directly, since FakeGCS's self-signed local cert would otherwise make
// the browser silently refuse to load the image.
func (h *CategoryHandler) serveThumbnail(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "category id is invalid")
		return
	}

	reader, contentType, err := h.service.OpenThumbnail(r.Context(), id)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}
	defer reader.Close()

	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	_, _ = io.Copy(w, reader)
}

func (h *CategoryHandler) deleteThumbnail(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "category id is invalid")
		return
	}

	category, err := h.service.DeleteThumbnail(r.Context(), id)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toCategoryResponse(*category))
}

func writeCatalogModuleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrCatalogNotFound):
		httpx.WriteError(w, http.StatusNotFound, "catalog_not_found", "catalog not found")
	case errors.Is(err, domain.ErrCategoryNotFound):
		httpx.WriteError(w, http.StatusNotFound, "category_not_found", "category not found")
	case errors.Is(err, domain.ErrProductTypeNotFound):
		httpx.WriteError(w, http.StatusNotFound, "product_type_not_found", "product type not found")
	case errors.Is(err, domain.ErrAttributeNotFound):
		httpx.WriteError(w, http.StatusNotFound, "attribute_not_found", "attribute not found")
	case errors.Is(err, domain.ErrAttributeValueNotFound):
		httpx.WriteError(w, http.StatusNotFound, "attribute_value_not_found", "attribute value not found")
	case errors.Is(err, domain.ErrProductNotFound):
		httpx.WriteError(w, http.StatusNotFound, "product_not_found", "product not found")
	case errors.Is(err, domain.ErrVariantNotFound):
		httpx.WriteError(w, http.StatusNotFound, "variant_not_found", "product variant not found")
	case errors.Is(err, domain.ErrMediaNotFound):
		httpx.WriteError(w, http.StatusNotFound, "media_not_found", "product media not found")
	case errors.Is(err, domain.ErrThumbnailNotFound):
		httpx.WriteError(w, http.StatusNotFound, "thumbnail_not_found", "category thumbnail not found")
	case errors.Is(err, domain.ErrInvalidStatus):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_status", "status is invalid")
	case errors.Is(err, application.ErrSlugConflict):
		httpx.WriteError(w, http.StatusConflict, "slug_conflict", "could not allocate a unique slug")
	case errors.As(err, new(domain.ValidationError)):
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", err.Error())
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
