package http

import (
	"context"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
	i18napplication "github.com/adotomov/fashion-store/apps/api/internal/modules/i18n/application"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

// Entity type tags used as the entity_type column in the shared translations
// table — the admin frontend calls PUT /admin/translations/{entityType}/{id}/{locale}
// with these exact strings to set per-locale overrides.
const (
	entityTypeProduct     = "product"
	entityTypeCategory    = "category"
	entityTypeCatalog     = "catalog"
	entityTypeProductType = "product_type"
	entityTypeAttribute   = "attribute"
	entityTypeAttrValue   = "attribute_value"
)

// EffectivePromoPrice is the storefront's view of an active promotion price
// for a product — decoupled from the promotions module's domain types.
type EffectivePromoPrice struct {
	Price money.Money
	Label string
}

// StorefrontPromotionsGateway is the minimal interface the storefront handler
// needs from the promotions module. BasePrices must include the base price for
// each product ID so the adapter can compute the discounted effective price
// without the storefront handler importing promotions domain types directly.
type StorefrontPromotionsGateway interface {
	GetEffectivePrices(ctx context.Context, productBasePrices map[uuid.UUID]money.Money) (map[uuid.UUID]EffectivePromoPrice, error)
}

// NavCategoryPromotionsGateway returns which of the given category IDs have
// at least one currently active promotion targeting them.
type NavCategoryPromotionsGateway interface {
	GetCategoriesWithActivePromotions(ctx context.Context, categoryIDs []uuid.UUID) (map[uuid.UUID]bool, error)
}

// StorefrontHandler exposes read-only, public (no admin auth) endpoints
// backing the customer-facing site: nav menu, product listings, and media
// file serving. It composes the same services the admin handlers use, but
// only ever reads, and always filters to status=active. When the request
// carries ?locale=<code> other than the default "en", translated fields are
// overlaid from the i18n module's translations table, falling back to the
// base English value for anything not yet translated.
type StorefrontHandler struct {
	productTypeService *application.ProductTypeService
	categoryService    *application.CategoryService
	productService     *application.ProductService
	catalogService     *application.CatalogService
	translations       *i18napplication.TranslationService
	promotions         StorefrontPromotionsGateway
	navPromos          NavCategoryPromotionsGateway
}

func NewStorefrontHandler(
	productTypeService *application.ProductTypeService,
	categoryService *application.CategoryService,
	productService *application.ProductService,
	catalogService *application.CatalogService,
	translations *i18napplication.TranslationService,
	promotions StorefrontPromotionsGateway,
	navPromos NavCategoryPromotionsGateway,
) *StorefrontHandler {
	return &StorefrontHandler{
		productTypeService: productTypeService,
		categoryService:    categoryService,
		productService:     productService,
		catalogService:     catalogService,
		translations:       translations,
		promotions:         promotions,
		navPromos:          navPromos,
	}
}

func (h *StorefrontHandler) RegisterRoutes(r chi.Router) {
	r.Route("/storefront", func(r chi.Router) {
		r.Get("/nav", h.nav)
		r.Get("/products", h.listProducts)
		r.Get("/products/best-in-category", h.bestInCategory)
		r.Get("/products/{slug}", h.getProduct)
		r.Get("/catalogs", h.listCatalogs)
		r.Get("/facets", h.facets)
		r.Get("/media/{mediaId}/file", h.serveMedia)
		r.Get("/categories/{categoryId}/thumbnail/file", h.serveCategoryThumbnail)
	})
}

func localeOf(r *http.Request) string {
	if locale := r.URL.Query().Get("locale"); locale != "" {
		return locale
	}
	return "en"
}

type navCategoryResponse struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Slug         string  `json:"slug"`
	ImageURL     *string `json:"image_url,omitempty"`
	HasPromotion bool    `json:"has_promotion"`
}

type navTypeResponse struct {
	ID         string                `json:"id"`
	Name       string                `json:"name"`
	Slug       string                `json:"slug"`
	Categories []navCategoryResponse `json:"categories"`
}

func (h *StorefrontHandler) nav(w http.ResponseWriter, r *http.Request) {
	productTypes, err := h.productTypeService.ListProductTypes(r.Context())
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}
	categories, err := h.categoryService.ListCategories(r.Context())
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	locale := localeOf(r)
	categoryTranslations, err := h.translations.ListByEntityType(r.Context(), entityTypeCategory, locale)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}
	typeTranslations, err := h.translations.ListByEntityType(r.Context(), entityTypeProductType, locale)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	categoriesByType := map[uuid.UUID][]navCategoryResponse{}
	for _, c := range categories {
		// Only top-level categories appear in the nav dropdown; subcategories
		// belong to a category page, not the mega-menu, to keep it scannable.
		if c.ParentID != nil {
			continue
		}
		name := c.Name
		if v, ok := categoryTranslations[c.ID]["name"]; ok {
			name = v
		}
		navCat := navCategoryResponse{ID: c.ID.String(), Name: name, Slug: c.Slug}
		if c.HasThumbnail() {
			url := "/api/v1/storefront/categories/" + c.ID.String() + "/thumbnail/file"
			navCat.ImageURL = &url
		}
		categoriesByType[c.ProductTypeID] = append(categoriesByType[c.ProductTypeID], navCat)
	}

	sort.Slice(productTypes, func(i, j int) bool {
		if productTypes[i].Position != productTypes[j].Position {
			return productTypes[i].Position < productTypes[j].Position
		}
		return productTypes[i].Name < productTypes[j].Name
	})

	resp := make([]navTypeResponse, 0, len(productTypes))
	for _, t := range productTypes {
		cats := categoriesByType[t.ID]
		if cats == nil {
			cats = []navCategoryResponse{}
		}
		name := t.Name
		if v, ok := typeTranslations[t.ID]["name"]; ok {
			name = v
		}
		resp = append(resp, navTypeResponse{
			ID:         t.ID.String(),
			Name:       name,
			Slug:       t.Slug,
			Categories: cats,
		})
	}

	// Annotate categories with active-promotion flags (best-effort).
	if h.navPromos != nil {
		var ids []uuid.UUID
		for i := range resp {
			for j := range resp[i].Categories {
				id, _ := uuid.Parse(resp[i].Categories[j].ID)
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			promoted, _ := h.navPromos.GetCategoriesWithActivePromotions(r.Context(), ids)
			for i := range resp {
				for j := range resp[i].Categories {
					id, _ := uuid.Parse(resp[i].Categories[j].ID)
					resp[i].Categories[j].HasPromotion = promoted[id]
				}
			}
		}
	}

	httpx.WriteJSON(w, http.StatusOK, resp)
}

type storefrontProductResponse struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Slug           string         `json:"slug"`
	Description    string         `json:"description"`
	BasePrice      moneyResponse  `json:"base_price"`
	CompareAtPrice *moneyResponse `json:"compare_at_price,omitempty"`
	// PromotionPrice is the effective discounted price when an active promotion
	// applies to this product. The frontend shows BasePrice struck through and
	// PromotionPrice as the actual selling price.
	PromotionPrice *moneyResponse            `json:"promotion_price,omitempty"`
	PromotionLabel *string                   `json:"promotion_label,omitempty"`
	ImageURL       *string                   `json:"image_url,omitempty"`
	Media          []storefrontMediaResponse `json:"media,omitempty"`
	Attributes     []attributeRefResponse    `json:"attributes,omitempty"`
	Variants       []variantResponse         `json:"variants,omitempty"`
	InStock        bool                      `json:"in_stock"`
	CreatedAt      string                    `json:"created_at"`
}

type storefrontMediaResponse struct {
	ID      string `json:"id"`
	URL     string `json:"url"`
	AltText string `json:"alt_text,omitempty"`
}

func toStorefrontProductResponse(p domain.Product) storefrontProductResponse {
	resp := storefrontProductResponse{
		ID:          p.ID.String(),
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		BasePrice:   toMoneyResponse(p.BasePrice),
		InStock:     p.InStock,
		CreatedAt:   p.CreatedAt.Format(timeFormat),
	}
	if p.CompareAtPrice != nil {
		compareAtPrice := toMoneyResponse(*p.CompareAtPrice)
		resp.CompareAtPrice = &compareAtPrice
	}
	if p.PrimaryMedia != nil {
		url := "/api/v1/storefront/media/" + p.PrimaryMedia.ID.String() + "/file"
		resp.ImageURL = &url
	}
	// Media/Attributes/Variants are only populated for FindByID/FindBySlug
	// (List() skips them for performance), so these are naturally omitted
	// from list responses.
	for _, m := range p.Media {
		resp.Media = append(resp.Media, storefrontMediaResponse{
			ID:      m.ID.String(),
			URL:     "/api/v1/storefront/media/" + m.ID.String() + "/file",
			AltText: m.AltText,
		})
	}
	for _, a := range p.Attributes {
		resp.Attributes = append(resp.Attributes, toAttributeRefResponse(a))
	}
	for _, v := range p.Variants {
		resp.Variants = append(resp.Variants, toVariantResponse(v))
	}
	return resp
}

// parseIDListParam collects every value of the given query key, splitting
// each on commas too, so callers can send either repeated `?key=a&key=b` or
// `?key=a,b` — the frontend uses repeated keys, but comma-separated stays
// supported for convenience.
func parseIDListParam(r *http.Request, key string) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	for _, raw := range r.URL.Query()[key] {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := uuid.Parse(part)
			if err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (h *StorefrontHandler) listProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.productService.ListProducts(r.Context())
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	// List() doesn't load per-product category/catalog assignments (it's
	// deliberately lightweight), so membership filters go through a
	// dedicated join-table query instead of inspecting product fields.
	var allowedIDs map[uuid.UUID]bool

	categoryIDs, err := parseIDListParam(r, "category_id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_category_id", "category_id is invalid")
		return
	}
	if len(categoryIDs) > 0 {
		// OR across categories: a product matching ANY selected category qualifies.
		categorySet := map[uuid.UUID]bool{}
		for _, categoryID := range categoryIDs {
			ids, err := h.productService.ProductIDsByCategory(r.Context(), categoryID)
			if err != nil {
				writeCatalogModuleError(w, err)
				return
			}
			for _, id := range ids {
				categorySet[id] = true
			}
		}
		allowedIDs = categorySet
	}

	if raw := r.URL.Query().Get("catalog_id"); raw != "" {
		catalogID, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_catalog_id", "catalog_id is invalid")
			return
		}
		ids, err := h.productService.ProductIDsByCatalog(r.Context(), catalogID)
		if err != nil {
			writeCatalogModuleError(w, err)
			return
		}
		allowedIDs = intersectOrSet(allowedIDs, toIDSet(ids))
	}

	attributeValueIDs, err := parseIDListParam(r, "attribute_value_id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_attribute_value_id", "attribute_value_id is invalid")
		return
	}
	if len(attributeValueIDs) > 0 {
		ids, err := h.productService.ProductIDsByAttributeValues(r.Context(), attributeValueIDs)
		if err != nil {
			writeCatalogModuleError(w, err)
			return
		}
		allowedIDs = intersectOrSet(allowedIDs, toIDSet(ids))
	}

	// Optional explicit product ID filter (for curated home sections).
	productIDFilter, err := parseIDListParam(r, "product_id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_product_id", "product_id is invalid")
		return
	}
	if len(productIDFilter) > 0 {
		allowedIDs = intersectOrSet(allowedIDs, toIDSet(productIDFilter))
	}

	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	hasPromotionOnly := r.URL.Query().Get("has_promotion") == "true"

	limit := -1
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}

	locale := localeOf(r)
	productTranslations, err := h.translations.ListByEntityType(r.Context(), entityTypeProduct, locale)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	// Collect active product IDs for promotion lookup.
	filtered := make([]domain.Product, 0, len(products))
	for _, p := range products {
		if p.Status != domain.ProductStatusActive {
			continue
		}
		if allowedIDs != nil && !allowedIDs[p.ID] {
			continue
		}
		filtered = append(filtered, p)
	}

	basePrices := make(map[uuid.UUID]money.Money, len(filtered))
	for _, p := range filtered {
		basePrices[p.ID] = p.BasePrice
	}
	var promos map[uuid.UUID]EffectivePromoPrice
	if h.promotions != nil {
		promos, _ = h.promotions.GetEffectivePrices(r.Context(), basePrices) // best-effort
	}

	resp := make([]storefrontProductResponse, 0, len(filtered))
	for _, p := range filtered {
		item := toStorefrontProductResponse(p)
		if v, ok := productTranslations[p.ID]["name"]; ok {
			item.Name = v
		}
		if v, ok := productTranslations[p.ID]["description"]; ok {
			item.Description = v
		}
		if query != "" &&
			!strings.Contains(strings.ToLower(item.Name), query) &&
			!strings.Contains(strings.ToLower(item.Description), query) {
			continue
		}
		if ep, ok := promos[p.ID]; ok {
			pr := toMoneyResponse(ep.Price)
			item.PromotionPrice = &pr
			item.PromotionLabel = &ep.Label
		}
		if hasPromotionOnly && item.PromotionPrice == nil {
			continue
		}
		resp = append(resp, item)
		if limit > 0 && len(resp) >= limit {
			break
		}
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *StorefrontHandler) bestInCategory(w http.ResponseWriter, r *http.Request) {
	ids, err := h.productService.BestInCategoryProductIDs(r.Context())
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}
	if len(ids) == 0 {
		httpx.WriteJSON(w, http.StatusOK, []storefrontProductResponse{})
		return
	}

	products, err := h.productService.ListProducts(r.Context())
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}
	idSet := toIDSet(ids)

	locale := localeOf(r)
	productTranslations, err := h.translations.ListByEntityType(r.Context(), entityTypeProduct, locale)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	var active []domain.Product
	for _, p := range products {
		if p.Status == domain.ProductStatusActive && idSet[p.ID] {
			active = append(active, p)
		}
	}

	basePrices := make(map[uuid.UUID]money.Money, len(active))
	for _, p := range active {
		basePrices[p.ID] = p.BasePrice
	}
	var promos map[uuid.UUID]EffectivePromoPrice
	if h.promotions != nil {
		promos, _ = h.promotions.GetEffectivePrices(r.Context(), basePrices)
	}

	resp := make([]storefrontProductResponse, 0, len(active))
	for _, p := range active {
		item := toStorefrontProductResponse(p)
		if v, ok := productTranslations[p.ID]["name"]; ok {
			item.Name = v
		}
		if v, ok := productTranslations[p.ID]["description"]; ok {
			item.Description = v
		}
		if ep, ok := promos[p.ID]; ok {
			pr := toMoneyResponse(ep.Price)
			item.PromotionPrice = &pr
			item.PromotionLabel = &ep.Label
		}
		resp = append(resp, item)
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func toIDSet(ids []uuid.UUID) map[uuid.UUID]bool {
	set := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

// intersectOrSet ANDs a new filter's matches into the running allowed set —
// if this is the first filter applied, its set becomes the baseline.
func intersectOrSet(existing, next map[uuid.UUID]bool) map[uuid.UUID]bool {
	if existing == nil {
		return next
	}
	for id := range existing {
		if !next[id] {
			delete(existing, id)
		}
	}
	return existing
}

type facetValueResponse struct {
	ID       string  `json:"id"`
	Value    string  `json:"value"`
	ColorHex *string `json:"color_hex,omitempty"`
}

type facetResponse struct {
	AttributeID   string               `json:"attribute_id"`
	AttributeName string               `json:"attribute_name"`
	AttributeType string               `json:"attribute_type"`
	Values        []facetValueResponse `json:"values"`
}

func (h *StorefrontHandler) facets(w http.ResponseWriter, r *http.Request) {
	categoryIDs, err := parseIDListParam(r, "category_id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_category_id", "category_id is invalid")
		return
	}

	var catalogID *uuid.UUID
	if raw := r.URL.Query().Get("catalog_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_catalog_id", "catalog_id is invalid")
			return
		}
		catalogID = &id
	}

	facets, err := h.productService.AttributeFacets(r.Context(), categoryIDs, catalogID)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	locale := localeOf(r)
	attrTranslations, err := h.translations.ListByEntityType(r.Context(), entityTypeAttribute, locale)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}
	valueTranslations, err := h.translations.ListByEntityType(r.Context(), entityTypeAttrValue, locale)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	resp := make([]facetResponse, 0, len(facets))
	for _, f := range facets {
		attributeName := f.AttributeName
		if v, ok := attrTranslations[f.AttributeID]["name"]; ok {
			attributeName = v
		}
		values := make([]facetValueResponse, 0, len(f.Values))
		for _, v := range f.Values {
			value := v.Value
			if t, ok := valueTranslations[v.ID]["value"]; ok {
				value = t
			}
			values = append(values, facetValueResponse{ID: v.ID.String(), Value: value, ColorHex: v.ColorHex})
		}
		resp = append(resp, facetResponse{
			AttributeID:   f.AttributeID.String(),
			AttributeName: attributeName,
			AttributeType: string(f.AttributeType),
			Values:        values,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *StorefrontHandler) getProduct(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	product, err := h.productService.GetProductBySlug(r.Context(), slug)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}
	if product.Status != domain.ProductStatusActive {
		httpx.WriteError(w, http.StatusNotFound, "product_not_found", "product not found")
		return
	}

	resp := toStorefrontProductResponse(*product)
	fields, err := h.translations.Get(r.Context(), entityTypeProduct, product.ID, localeOf(r))
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}
	if v, ok := fields["name"]; ok {
		resp.Name = v
	}
	if v, ok := fields["description"]; ok {
		resp.Description = v
	}
	if h.promotions != nil {
		if promos, _ := h.promotions.GetEffectivePrices(r.Context(), map[uuid.UUID]money.Money{product.ID: product.BasePrice}); promos != nil {
			if ep, ok := promos[product.ID]; ok {
				pr := toMoneyResponse(ep.Price)
				resp.PromotionPrice = &pr
				resp.PromotionLabel = &ep.Label
			}
		}
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

type storefrontCatalogResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (h *StorefrontHandler) listCatalogs(w http.ResponseWriter, r *http.Request) {
	catalogs, err := h.catalogService.ListCatalogs(r.Context())
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	locale := localeOf(r)
	catalogTranslations, err := h.translations.ListByEntityType(r.Context(), entityTypeCatalog, locale)
	if err != nil {
		writeCatalogModuleError(w, err)
		return
	}

	resp := make([]storefrontCatalogResponse, 0, len(catalogs))
	for _, c := range catalogs {
		if c.Status != domain.StatusActive {
			continue
		}
		name := c.Name
		if v, ok := catalogTranslations[c.ID]["name"]; ok {
			name = v
		}
		resp = append(resp, storefrontCatalogResponse{ID: c.ID.String(), Name: name, Slug: c.Slug})
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

// serveMedia is the public counterpart to ProductHandler.serveMedia — same
// streaming logic, just mounted without requireAdmin since product imagery
// must be visible to anonymous storefront visitors.
func (h *StorefrontHandler) serveMedia(w http.ResponseWriter, r *http.Request) {
	mediaID, err := uuid.Parse(chi.URLParam(r, "mediaId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "media id is invalid")
		return
	}

	reader, contentType, err := h.productService.OpenMedia(r.Context(), mediaID)
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

// serveCategoryThumbnail is the public counterpart to
// CategoryHandler.serveThumbnail — same streaming logic, just mounted
// without requireAdmin since nav-menu imagery must be visible to
// anonymous storefront visitors.
func (h *StorefrontHandler) serveCategoryThumbnail(w http.ResponseWriter, r *http.Request) {
	categoryID, err := uuid.Parse(chi.URLParam(r, "categoryId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "category id is invalid")
		return
	}

	reader, contentType, err := h.categoryService.OpenThumbnail(r.Context(), categoryID)
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
