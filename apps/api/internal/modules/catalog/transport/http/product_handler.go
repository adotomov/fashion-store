package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

type ProductHandler struct {
	service *application.ProductService
}

func NewProductHandler(service *application.ProductService) *ProductHandler {
	return &ProductHandler{service: service}
}

func (h *ProductHandler) RegisterRoutes(r chi.Router, requireAdmin func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(requireAdmin)
		r.Get("/admin/catalog/stats", h.stats)
		r.Route("/admin/products", func(r chi.Router) {
			r.Get("/", h.list)
			r.Post("/", h.create)
			r.Get("/{id}", h.get)
			r.Patch("/{id}", h.update)
			r.Delete("/{id}", h.delete)
			r.Put("/{id}/categories", h.setCategories)
			r.Put("/{id}/catalogs", h.setCatalogs)
			r.Put("/{id}/attributes", h.setAttributes)
			r.Post("/{id}/variants", h.createVariant)
			r.Patch("/{id}/variants/{variantId}", h.updateVariant)
			r.Delete("/{id}/variants/{variantId}", h.deleteVariant)
			r.Post("/{id}/media", h.createMedia)
			r.Get("/{id}/media/{mediaId}/file", h.serveMedia)
			r.Patch("/{id}/media/{mediaId}", h.updateMedia)
			r.Delete("/{id}/media/{mediaId}", h.deleteMedia)
		})
	})
}

type moneyResponse struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

func toMoneyResponse(m money.Money) moneyResponse {
	return moneyResponse{AmountMinor: m.AmountMinor, Currency: m.Currency}
}

type attributeRefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type productResponse struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Slug           string                 `json:"slug"`
	Description    string                 `json:"description"`
	Status         string                 `json:"status"`
	BasePrice      moneyResponse          `json:"base_price"`
	CompareAtPrice *moneyResponse         `json:"compare_at_price,omitempty"`
	NKSCode        string                 `json:"nks_code"`
	CategoryIDs    []string               `json:"category_ids,omitempty"`
	CatalogIDs     []string               `json:"catalog_ids,omitempty"`
	Attributes     []attributeRefResponse `json:"attributes,omitempty"`
	VariantCount   int                    `json:"variant_count"`
	Variants       []variantResponse      `json:"variants,omitempty"`
	Media          []mediaResponse        `json:"media,omitempty"`
	CreatedAt      string                 `json:"created_at"`
	UpdatedAt      string                 `json:"updated_at"`
}

func toProductResponse(p domain.Product) productResponse {
	resp := productResponse{
		ID:           p.ID.String(),
		Name:         p.Name,
		Slug:         p.Slug,
		Description:  p.Description,
		Status:       string(p.Status),
		BasePrice:    toMoneyResponse(p.BasePrice),
		NKSCode:      p.NKSCode,
		VariantCount: p.VariantCount,
		CreatedAt:    p.CreatedAt.Format(timeFormat),
		UpdatedAt:    p.UpdatedAt.Format(timeFormat),
	}
	if p.CompareAtPrice != nil {
		compareAtPrice := toMoneyResponse(*p.CompareAtPrice)
		resp.CompareAtPrice = &compareAtPrice
	}
	for _, id := range p.CategoryIDs {
		resp.CategoryIDs = append(resp.CategoryIDs, id.String())
	}
	for _, id := range p.CatalogIDs {
		resp.CatalogIDs = append(resp.CatalogIDs, id.String())
	}
	for _, a := range p.Attributes {
		resp.Attributes = append(resp.Attributes, attributeRefResponse{ID: a.ID.String(), Name: a.Name})
	}
	for _, v := range p.Variants {
		resp.Variants = append(resp.Variants, toVariantResponse(v))
	}
	for _, m := range p.Media {
		resp.Media = append(resp.Media, toMediaResponse(m))
	}
	return resp
}

func (h *ProductHandler) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	product, err := h.service.CreateProduct(r.Context(), application.CreateProductInput{Name: req.Name})
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toProductResponse(*product))
}

func (h *ProductHandler) list(w http.ResponseWriter, r *http.Request) {
	products, err := h.service.ListProducts(r.Context())
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	resp := make([]productResponse, 0, len(products))
	for _, p := range products {
		resp = append(resp, toProductResponse(p))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *ProductHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "product id is invalid")
		return
	}

	product, err := h.service.GetProduct(r.Context(), id)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toProductResponse(*product))
}

type updateProductRequest struct {
	Name                *string        `json:"name,omitempty"`
	Description         *string        `json:"description,omitempty"`
	Status              *string        `json:"status,omitempty"`
	BasePrice           *moneyResponse `json:"base_price,omitempty"`
	CompareAtPrice      *moneyResponse `json:"compare_at_price,omitempty"`
	ClearCompareAtPrice bool           `json:"clear_compare_at_price,omitempty"`
	NKSCode             *string        `json:"nks_code,omitempty"`
}

func (h *ProductHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "product id is invalid")
		return
	}

	var req updateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	input := application.UpdateProductInput{Name: req.Name, Description: req.Description, NKSCode: req.NKSCode}
	if req.Status != nil {
		status := domain.ProductStatus(*req.Status)
		input.Status = &status
	}
	if req.BasePrice != nil {
		input.BasePrice = &money.Money{AmountMinor: req.BasePrice.AmountMinor, Currency: req.BasePrice.Currency}
	}
	if req.ClearCompareAtPrice {
		input.ClearCompareAtPrice = true
	} else if req.CompareAtPrice != nil {
		input.CompareAtPrice = &money.Money{AmountMinor: req.CompareAtPrice.AmountMinor, Currency: req.CompareAtPrice.Currency}
	}

	product, err := h.service.UpdateProduct(r.Context(), id, input)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toProductResponse(*product))
}

func (h *ProductHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "product id is invalid")
		return
	}

	if err := h.service.DeleteProduct(r.Context(), id); err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func decodeIDList(r *http.Request, field string) ([]uuid.UUID, error) {
	var req map[string][]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	raw := req[field]
	ids := make([]uuid.UUID, 0, len(raw))
	for _, s := range raw {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (h *ProductHandler) setCategories(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "product id is invalid")
		return
	}
	ids, err := decodeIDList(r, "category_ids")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "category_ids is invalid")
		return
	}
	if err := h.service.SetCategories(r.Context(), id, ids); err != nil {
		writeCatalogModuleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProductHandler) setCatalogs(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "product id is invalid")
		return
	}
	ids, err := decodeIDList(r, "catalog_ids")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "catalog_ids is invalid")
		return
	}
	if err := h.service.SetCatalogs(r.Context(), id, ids); err != nil {
		writeCatalogModuleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type topProductResponse struct {
	ProductID    string `json:"product_id"`
	ProductName  string `json:"product_name"`
	QuantitySold int    `json:"quantity_sold"`
	OrderCount   int    `json:"order_count"`
}

type catalogStatsResponse struct {
	TotalProducts    int                  `json:"total_products"`
	ActiveProducts   int                  `json:"active_products"`
	DraftProducts    int                  `json:"draft_products"`
	ArchivedProducts int                  `json:"archived_products"`
	TotalVariants    int                  `json:"total_variants"`
	TotalCategories  int                  `json:"total_categories"`
	TopProducts      []topProductResponse `json:"top_products"`
}

func (h *ProductHandler) stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.CatalogStats(r.Context())
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	resp := catalogStatsResponse{
		TotalProducts:    stats.TotalProducts,
		ActiveProducts:   stats.ActiveProducts,
		DraftProducts:    stats.DraftProducts,
		ArchivedProducts: stats.ArchivedProducts,
		TotalVariants:    stats.TotalVariants,
		TotalCategories:  stats.TotalCategories,
		TopProducts:      make([]topProductResponse, 0, len(stats.TopProducts)),
	}
	for _, tp := range stats.TopProducts {
		resp.TopProducts = append(resp.TopProducts, topProductResponse{
			ProductID:    tp.ProductID.String(),
			ProductName:  tp.ProductName,
			QuantitySold: tp.QuantitySold,
			OrderCount:   tp.OrderCount,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *ProductHandler) setAttributes(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "product id is invalid")
		return
	}
	ids, err := decodeIDList(r, "attribute_ids")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "attribute_ids is invalid")
		return
	}
	if err := h.service.SetAttributes(r.Context(), id, ids); err != nil {
		writeCatalogModuleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
