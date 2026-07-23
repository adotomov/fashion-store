package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	catalogapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
	cataloginfra "github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/infrastructure"
	cataloghttp "github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/transport/http"
	i18napplication "github.com/adotomov/fashion-store/apps/api/internal/modules/i18n/application"
	i18ninfra "github.com/adotomov/fashion-store/apps/api/internal/modules/i18n/infrastructure"
)

func TestStorefrontHTTP_ProductsAndFacetsFilters(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping Postgres integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	t.Cleanup(pool.Close)

	categoryRepo := cataloginfra.NewPostgresCategoryRepository(pool)
	productTypeRepo := cataloginfra.NewPostgresProductTypeRepository(pool)
	attributeRepo := cataloginfra.NewPostgresAttributeRepository(pool)
	productRepo := cataloginfra.NewPostgresProductRepository(pool)
	catalogRepo := cataloginfra.NewPostgresCatalogRepository(pool)

	categoryService := catalogapplication.NewCategoryService(categoryRepo, nil, "")
	productTypeService := catalogapplication.NewProductTypeService(productTypeRepo)
	attributeService := catalogapplication.NewAttributeService(attributeRepo)
	productService := catalogapplication.NewProductService(productRepo, nil, "")
	catalogService := catalogapplication.NewCatalogService(catalogRepo)
	translationService := i18napplication.NewTranslationService(i18ninfra.NewPostgresTranslationRepository(pool))

	storefrontHandler := cataloghttp.NewStorefrontHandler(productTypeService, categoryService, productService, catalogService, translationService, nil, nil, nil)
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		storefrontHandler.RegisterRoutes(r)
	})

	ctx := context.Background()

	productType, err := productTypeService.CreateProductType(ctx, catalogapplication.CreateProductTypeInput{Name: "HTTP-Test Shop Type"})
	if err != nil {
		t.Fatalf("create product type: %v", err)
	}
	t.Cleanup(func() { _ = productTypeService.DeleteProductType(ctx, productType.ID) })

	catA, err := categoryService.CreateCategory(ctx, catalogapplication.CreateCategoryInput{Name: "HTTP-Test Shop Cat A", ProductTypeID: productType.ID})
	if err != nil {
		t.Fatalf("create category A: %v", err)
	}
	t.Cleanup(func() { _ = categoryService.DeleteCategory(ctx, catA.ID) })

	catB, err := categoryService.CreateCategory(ctx, catalogapplication.CreateCategoryInput{Name: "HTTP-Test Shop Cat B", ProductTypeID: productType.ID})
	if err != nil {
		t.Fatalf("create category B: %v", err)
	}
	t.Cleanup(func() { _ = categoryService.DeleteCategory(ctx, catB.ID) })

	sizeAttr, err := attributeService.CreateAttribute(ctx, catalogapplication.CreateAttributeInput{Name: "HTTP-Test Shop Size"})
	if err != nil {
		t.Fatalf("create attribute: %v", err)
	}
	t.Cleanup(func() { _ = attributeService.DeleteAttribute(ctx, sizeAttr.ID) })
	sizeS, err := attributeService.AddValue(ctx, sizeAttr.ID, "S", nil)
	if err != nil {
		t.Fatalf("add value S: %v", err)
	}

	productInA, err := productService.CreateProduct(ctx, catalogapplication.CreateProductInput{Name: "HTTP-Test Shop Product A"})
	if err != nil {
		t.Fatalf("create product A: %v", err)
	}
	t.Cleanup(func() { _ = productService.DeleteProduct(ctx, productInA.ID) })
	active := "active"
	if _, err := productService.UpdateProduct(ctx, productInA.ID, catalogapplication.UpdateProductInput{Status: statusPtr(active)}); err != nil {
		t.Fatalf("activate product A: %v", err)
	}
	if err := productService.SetCategories(ctx, productInA.ID, idSlice(catA.ID)); err != nil {
		t.Fatalf("set categories A: %v", err)
	}
	if _, err := productService.AddVariant(ctx, productInA.ID, catalogapplication.CreateVariantInput{AttributeValueIDs: idSlice(sizeS.ID)}); err != nil {
		t.Fatalf("add variant A: %v", err)
	}

	productInB, err := productService.CreateProduct(ctx, catalogapplication.CreateProductInput{Name: "HTTP-Test Shop Product B"})
	if err != nil {
		t.Fatalf("create product B: %v", err)
	}
	t.Cleanup(func() { _ = productService.DeleteProduct(ctx, productInB.ID) })
	if _, err := productService.UpdateProduct(ctx, productInB.ID, catalogapplication.UpdateProductInput{Status: statusPtr(active)}); err != nil {
		t.Fatalf("activate product B: %v", err)
	}
	if err := productService.SetCategories(ctx, productInB.ID, idSlice(catB.ID)); err != nil {
		t.Fatalf("set categories B: %v", err)
	}

	// Multi-category OR: both categories selected should return both products.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/storefront/products?category_id="+catA.ID.String()+"&category_id="+catB.ID.String(), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list products: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var productPage struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &productPage); err != nil {
		t.Fatalf("decode products: %v", err)
	}
	if len(productPage.Items) != 2 || productPage.Total != 2 {
		t.Errorf("expected 2 products for category A OR B, got %d (total %d)", len(productPage.Items), productPage.Total)
	}

	// Attribute filter: only product A has the Size=S variant.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/storefront/products?attribute_value_id="+sizeS.ID.String(), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list products by attribute: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &productPage); err != nil {
		t.Fatalf("decode products: %v", err)
	}
	if len(productPage.Items) != 1 || productPage.Items[0].ID != productInA.ID.String() {
		t.Errorf("expected only product A for Size=S, got %v", productPage.Items)
	}

	// Facets scoped to category A should include the Size attribute/value.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/storefront/facets?category_id="+catA.ID.String(), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("facets: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var facets []struct {
		AttributeName string `json:"attribute_name"`
		Values        []struct {
			Value string `json:"value"`
		} `json:"values"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &facets); err != nil {
		t.Fatalf("decode facets: %v", err)
	}
	found := false
	for _, f := range facets {
		if f.AttributeName == sizeAttr.Name {
			for _, v := range f.Values {
				if v.Value == "S" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Errorf("expected Size=S facet for category A, got %+v", facets)
	}
}

func TestStorefrontHTTP_GetProductBySlugIncludesVariantsAndMedia(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping Postgres integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	t.Cleanup(pool.Close)

	attributeRepo := cataloginfra.NewPostgresAttributeRepository(pool)
	productRepo := cataloginfra.NewPostgresProductRepository(pool)
	catalogRepo := cataloginfra.NewPostgresCatalogRepository(pool)
	categoryRepo := cataloginfra.NewPostgresCategoryRepository(pool)
	productTypeRepo := cataloginfra.NewPostgresProductTypeRepository(pool)

	attributeService := catalogapplication.NewAttributeService(attributeRepo)
	productService := catalogapplication.NewProductService(productRepo, nil, "")
	catalogService := catalogapplication.NewCatalogService(catalogRepo)
	categoryService := catalogapplication.NewCategoryService(categoryRepo, nil, "")
	productTypeService := catalogapplication.NewProductTypeService(productTypeRepo)
	translationService := i18napplication.NewTranslationService(i18ninfra.NewPostgresTranslationRepository(pool))

	storefrontHandler := cataloghttp.NewStorefrontHandler(productTypeService, categoryService, productService, catalogService, translationService, nil, nil, nil)
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		storefrontHandler.RegisterRoutes(r)
	})

	ctx := context.Background()

	sizeAttr, err := attributeService.CreateAttribute(ctx, catalogapplication.CreateAttributeInput{Name: "HTTP-Test Slug Size"})
	if err != nil {
		t.Fatalf("create attribute: %v", err)
	}
	t.Cleanup(func() { _ = attributeService.DeleteAttribute(ctx, sizeAttr.ID) })
	sizeM, err := attributeService.AddValue(ctx, sizeAttr.ID, "M", nil)
	if err != nil {
		t.Fatalf("add value M: %v", err)
	}

	product, err := productService.CreateProduct(ctx, catalogapplication.CreateProductInput{Name: "HTTP-Test Slug Product"})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	t.Cleanup(func() { _ = productService.DeleteProduct(ctx, product.ID) })
	active := "active"
	if _, err := productService.UpdateProduct(ctx, product.ID, catalogapplication.UpdateProductInput{Status: statusPtr(active)}); err != nil {
		t.Fatalf("activate product: %v", err)
	}
	if _, err := productService.AddVariant(ctx, product.ID, catalogapplication.CreateVariantInput{AttributeValueIDs: idSlice(sizeM.ID)}); err != nil {
		t.Fatalf("add variant: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/storefront/products/"+product.Slug, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get product by slug: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var got struct {
		ID       string `json:"id"`
		Variants []struct {
			Attributes []struct {
				Value string `json:"value"`
			} `json:"attributes"`
		} `json:"variants"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode product: %v", err)
	}
	if got.ID != product.ID.String() {
		t.Fatalf("expected product id %s, got %s", product.ID, got.ID)
	}
	if len(got.Variants) != 1 || len(got.Variants[0].Attributes) != 1 || got.Variants[0].Attributes[0].Value != "M" {
		t.Fatalf("expected 1 variant with Size=M, got %+v", got.Variants)
	}

	// Unknown slug returns 404, not a 400 (no uuid parsing involved anymore).
	req = httptest.NewRequest(http.MethodGet, "/api/v1/storefront/products/does-not-exist", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown slug, got %d: %s", rec.Code, rec.Body.String())
	}
}

func statusPtr(s string) *domain.ProductStatus {
	status := domain.ProductStatus(s)
	return &status
}

func idSlice(id uuid.UUID) []uuid.UUID {
	return []uuid.UUID{id}
}
