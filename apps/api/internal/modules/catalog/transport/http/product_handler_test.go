package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	catalogapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/application"
	cataloginfra "github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/infrastructure"
	cataloghttp "github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/transport/http"
)

// noop passes every request through unauthenticated — these tests exercise
// only the routing/JSON/repo path, not the auth middleware (covered
// elsewhere).
func noopMiddleware(next http.Handler) http.Handler { return next }

func TestProductHTTP_AssignMultipleCategoriesAndCatalogs(t *testing.T) {
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
	productRepo := cataloginfra.NewPostgresProductRepository(pool)
	catalogRepo := cataloginfra.NewPostgresCatalogRepository(pool)

	categoryService := catalogapplication.NewCategoryService(categoryRepo, nil, "")
	productTypeService := catalogapplication.NewProductTypeService(productTypeRepo)
	productService := catalogapplication.NewProductService(productRepo, nil, "")
	catalogService := catalogapplication.NewCatalogService(catalogRepo)

	productHandler := cataloghttp.NewProductHandler(productService)
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		productHandler.RegisterRoutes(r, noopMiddleware)
	})

	ctx := context.Background()

	productType, err := productTypeService.CreateProductType(ctx, catalogapplication.CreateProductTypeInput{Name: "HTTP-Test Type"})
	if err != nil {
		t.Fatalf("create product type: %v", err)
	}
	t.Cleanup(func() { _ = productTypeService.DeleteProductType(ctx, productType.ID) })

	cat1, err := categoryService.CreateCategory(ctx, catalogapplication.CreateCategoryInput{Name: "HTTP-Test Cat 1", ProductTypeID: productType.ID})
	if err != nil {
		t.Fatalf("create category 1: %v", err)
	}
	t.Cleanup(func() { _ = categoryService.DeleteCategory(ctx, cat1.ID) })

	cat2, err := categoryService.CreateCategory(ctx, catalogapplication.CreateCategoryInput{Name: "HTTP-Test Cat 2", ProductTypeID: productType.ID})
	if err != nil {
		t.Fatalf("create category 2: %v", err)
	}
	t.Cleanup(func() { _ = categoryService.DeleteCategory(ctx, cat2.ID) })

	catalog1, err := catalogService.CreateCatalog(ctx, catalogapplication.CreateCatalogInput{Name: "HTTP-Test Catalog 1"})
	if err != nil {
		t.Fatalf("create catalog 1: %v", err)
	}
	t.Cleanup(func() { _ = catalogService.DeleteCatalog(ctx, catalog1.ID) })

	catalog2, err := catalogService.CreateCatalog(ctx, catalogapplication.CreateCatalogInput{Name: "HTTP-Test Catalog 2"})
	if err != nil {
		t.Fatalf("create catalog 2: %v", err)
	}
	t.Cleanup(func() { _ = catalogService.DeleteCatalog(ctx, catalog2.ID) })

	product, err := productService.CreateProduct(ctx, catalogapplication.CreateProductInput{Name: "HTTP-Test Product"})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	t.Cleanup(func() { _ = productService.DeleteProduct(ctx, product.ID) })

	categoriesBody := `{"category_ids":["` + cat1.ID.String() + `","` + cat2.ID.String() + `"]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/products/"+product.ID.String()+"/categories", strings.NewReader(categoriesBody))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("set categories: expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	catalogsBody := `{"catalog_ids":["` + catalog1.ID.String() + `","` + catalog2.ID.String() + `"]}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/admin/products/"+product.ID.String()+"/catalogs", strings.NewReader(catalogsBody))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("set catalogs: expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/products/"+product.ID.String(), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get product: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var got struct {
		CategoryIDs []string `json:"category_ids"`
		CatalogIDs  []string `json:"catalog_ids"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(got.CategoryIDs) != 2 {
		t.Errorf("expected 2 category_ids in response, got %v", got.CategoryIDs)
	}
	if len(got.CatalogIDs) != 2 {
		t.Errorf("expected 2 catalog_ids in response, got %v", got.CatalogIDs)
	}
}

func TestProductHTTP_ListReportsVariantCount(t *testing.T) {
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

	attributeService := catalogapplication.NewAttributeService(attributeRepo)
	productService := catalogapplication.NewProductService(productRepo, nil, "")

	productHandler := cataloghttp.NewProductHandler(productService)
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		productHandler.RegisterRoutes(r, noopMiddleware)
	})

	ctx := context.Background()

	withVariant, err := productService.CreateProduct(ctx, catalogapplication.CreateProductInput{Name: "HTTP-Test Has Variant"})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	t.Cleanup(func() { _ = productService.DeleteProduct(ctx, withVariant.ID) })

	withoutVariant, err := productService.CreateProduct(ctx, catalogapplication.CreateProductInput{Name: "HTTP-Test No Variant"})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	t.Cleanup(func() { _ = productService.DeleteProduct(ctx, withoutVariant.ID) })

	attribute, err := attributeService.CreateAttribute(ctx, catalogapplication.CreateAttributeInput{Name: "HTTP-Test Size"})
	if err != nil {
		t.Fatalf("create attribute: %v", err)
	}
	t.Cleanup(func() { _ = attributeService.DeleteAttribute(ctx, attribute.ID) })

	value, err := attributeService.AddValue(ctx, attribute.ID, "S")
	if err != nil {
		t.Fatalf("add attribute value: %v", err)
	}

	if _, err := productService.AddVariant(ctx, withVariant.ID, catalogapplication.CreateVariantInput{
		AttributeValueIDs: []uuid.UUID{value.ID},
	}); err != nil {
		t.Fatalf("add variant: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/products", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list products: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var list []struct {
		ID           string `json:"id"`
		VariantCount int    `json:"variant_count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	counts := map[string]int{}
	for _, p := range list {
		counts[p.ID] = p.VariantCount
	}

	if counts[withVariant.ID.String()] != 1 {
		t.Errorf("expected variant_count 1 for product with a variant, got %d", counts[withVariant.ID.String()])
	}
	if counts[withoutVariant.ID.String()] != 0 {
		t.Errorf("expected variant_count 0 for product without variants, got %d", counts[withoutVariant.ID.String()])
	}
}
