package infrastructure_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/infrastructure"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

// Integration test against a real Postgres instance. Skips automatically if
// DATABASE_URL isn't set, per 23-testing-guidelines.md.
func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping Postgres integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestPostgresProductRepository_FullLifecycle(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	categoryRepo := infrastructure.NewPostgresCategoryRepository(pool)
	productTypeRepo := infrastructure.NewPostgresProductTypeRepository(pool)
	attributeRepo := infrastructure.NewPostgresAttributeRepository(pool)
	productRepo := infrastructure.NewPostgresProductRepository(pool)

	productType, err := productTypeRepo.Create(ctx, domain.ProductType{Name: "IT-Test Apparel", Slug: "it-test-apparel"})
	if err != nil {
		t.Fatalf("create product type: %v", err)
	}
	t.Cleanup(func() { _ = productTypeRepo.Delete(ctx, productType.ID) })

	category, err := categoryRepo.Create(ctx, domain.Category{Name: "IT-Test Dresses", Slug: "it-test-dresses", ProductTypeID: productType.ID})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	t.Cleanup(func() { _ = categoryRepo.Delete(ctx, category.ID) })

	sizeAttr, err := attributeRepo.Create(ctx, domain.Attribute{Name: "IT-Test Size"})
	if err != nil {
		t.Fatalf("create attribute: %v", err)
	}
	t.Cleanup(func() { _ = attributeRepo.Delete(ctx, sizeAttr.ID) })

	sizeS, err := attributeRepo.AddValue(ctx, sizeAttr.ID, "S", nil)
	if err != nil {
		t.Fatalf("add attribute value: %v", err)
	}
	sizeM, err := attributeRepo.AddValue(ctx, sizeAttr.ID, "M", nil)
	if err != nil {
		t.Fatalf("add attribute value: %v", err)
	}

	product, err := productRepo.Create(ctx, domain.Product{
		Name:      "IT-Test Silk Dress",
		Slug:      "it-test-silk-dress",
		Status:    domain.ProductStatusDraft,
		BasePrice: money.Money{AmountMinor: 8999, Currency: "EUR"},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	t.Cleanup(func() { _ = productRepo.Delete(ctx, product.ID) })

	category2, err := categoryRepo.Create(ctx, domain.Category{Name: "IT-Test Outerwear", Slug: "it-test-outerwear", ProductTypeID: productType.ID})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	t.Cleanup(func() { _ = categoryRepo.Delete(ctx, category2.ID) })

	if err := productRepo.SetCategories(ctx, product.ID, []uuid.UUID{category.ID, category2.ID}); err != nil {
		t.Fatalf("set categories: %v", err)
	}

	if err := productRepo.SetAttributes(ctx, product.ID, []uuid.UUID{sizeAttr.ID}); err != nil {
		t.Fatalf("set attributes: %v", err)
	}

	variant, err := productRepo.CreateVariant(ctx, domain.ProductVariant{ProductID: product.ID}, []uuid.UUID{sizeS.ID})
	if err != nil {
		t.Fatalf("create variant: %v", err)
	}
	if len(variant.Attributes) != 1 || variant.Attributes[0].Value != "S" {
		t.Fatalf("expected variant to have Size=S, got %+v", variant.Attributes)
	}

	media, err := productRepo.CreateMedia(ctx, domain.ProductMedia{
		ProductID: product.ID,
		Bucket:    "test-bucket",
		ObjectKey: "test-object.jpg",
		Position:  0,
	})
	if err != nil {
		t.Fatalf("create media: %v", err)
	}

	loaded, err := productRepo.FindByID(ctx, product.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if len(loaded.CategoryIDs) != 2 {
		t.Errorf("expected 2 categories, got %v", loaded.CategoryIDs)
	}
	if len(loaded.Attributes) != 1 || loaded.Attributes[0].ID != sizeAttr.ID || loaded.Attributes[0].Name != sizeAttr.Name {
		t.Errorf("expected product to reference Size attribute, got %+v", loaded.Attributes)
	}
	if len(loaded.Variants) != 1 {
		t.Errorf("expected 1 variant, got %d", len(loaded.Variants))
	}
	if len(loaded.Media) != 1 || loaded.Media[0].ID != media.ID {
		t.Errorf("expected 1 media item, got %v", loaded.Media)
	}

	updatedVariant, err := productRepo.UpdateVariant(ctx, *variant, []uuid.UUID{sizeM.ID})
	if err != nil {
		t.Fatalf("update variant: %v", err)
	}
	if len(updatedVariant.Attributes) != 1 || updatedVariant.Attributes[0].Value != "M" {
		t.Fatalf("expected variant attributes replaced with Size=M, got %+v", updatedVariant.Attributes)
	}

	if err := productRepo.DeleteVariant(ctx, variant.ID); err != nil {
		t.Fatalf("delete variant: %v", err)
	}
	if err := productRepo.DeleteMedia(ctx, media.ID); err != nil {
		t.Fatalf("delete media: %v", err)
	}
	if err := attributeRepo.DeleteValue(ctx, sizeAttr.ID, sizeM.ID); err != nil {
		t.Fatalf("delete attribute value: %v", err)
	}
}

func TestPostgresProductRepository_AttributeFacetsAndFiltering(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	categoryRepo := infrastructure.NewPostgresCategoryRepository(pool)
	productTypeRepo := infrastructure.NewPostgresProductTypeRepository(pool)
	attributeRepo := infrastructure.NewPostgresAttributeRepository(pool)
	productRepo := infrastructure.NewPostgresProductRepository(pool)

	productType, err := productTypeRepo.Create(ctx, domain.ProductType{Name: "IT-Test Facets Type", Slug: "it-test-facets-type"})
	if err != nil {
		t.Fatalf("create product type: %v", err)
	}
	t.Cleanup(func() { _ = productTypeRepo.Delete(ctx, productType.ID) })

	category, err := categoryRepo.Create(ctx, domain.Category{Name: "IT-Test Facets Category", Slug: "it-test-facets-category", ProductTypeID: productType.ID})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	t.Cleanup(func() { _ = categoryRepo.Delete(ctx, category.ID) })

	sizeAttr, err := attributeRepo.Create(ctx, domain.Attribute{Name: "IT-Test Facets Size"})
	if err != nil {
		t.Fatalf("create size attribute: %v", err)
	}
	t.Cleanup(func() { _ = attributeRepo.Delete(ctx, sizeAttr.ID) })
	sizeS, err := attributeRepo.AddValue(ctx, sizeAttr.ID, "S", nil)
	if err != nil {
		t.Fatalf("add size S: %v", err)
	}
	sizeM, err := attributeRepo.AddValue(ctx, sizeAttr.ID, "M", nil)
	if err != nil {
		t.Fatalf("add size M: %v", err)
	}

	colorAttr, err := attributeRepo.Create(ctx, domain.Attribute{Name: "IT-Test Facets Color"})
	if err != nil {
		t.Fatalf("create color attribute: %v", err)
	}
	t.Cleanup(func() { _ = attributeRepo.Delete(ctx, colorAttr.ID) })
	colorBlue, err := attributeRepo.AddValue(ctx, colorAttr.ID, "Blue", nil)
	if err != nil {
		t.Fatalf("add color blue: %v", err)
	}
	colorRed, err := attributeRepo.AddValue(ctx, colorAttr.ID, "Red", nil)
	if err != nil {
		t.Fatalf("add color red: %v", err)
	}

	productA, err := productRepo.Create(ctx, domain.Product{
		Name: "IT-Test Facets Product A", Slug: "it-test-facets-product-a",
		Status: domain.ProductStatusActive, BasePrice: money.Money{AmountMinor: 1000, Currency: "EUR"},
	})
	if err != nil {
		t.Fatalf("create product A: %v", err)
	}
	t.Cleanup(func() { _ = productRepo.Delete(ctx, productA.ID) })
	if err := productRepo.SetCategories(ctx, productA.ID, []uuid.UUID{category.ID}); err != nil {
		t.Fatalf("set categories A: %v", err)
	}
	if _, err := productRepo.CreateVariant(ctx, domain.ProductVariant{ProductID: productA.ID}, []uuid.UUID{sizeS.ID, colorBlue.ID}); err != nil {
		t.Fatalf("create variant A: %v", err)
	}

	productB, err := productRepo.Create(ctx, domain.Product{
		Name: "IT-Test Facets Product B", Slug: "it-test-facets-product-b",
		Status: domain.ProductStatusActive, BasePrice: money.Money{AmountMinor: 2000, Currency: "EUR"},
	})
	if err != nil {
		t.Fatalf("create product B: %v", err)
	}
	t.Cleanup(func() { _ = productRepo.Delete(ctx, productB.ID) })
	if err := productRepo.SetCategories(ctx, productB.ID, []uuid.UUID{category.ID}); err != nil {
		t.Fatalf("set categories B: %v", err)
	}
	if _, err := productRepo.CreateVariant(ctx, domain.ProductVariant{ProductID: productB.ID}, []uuid.UUID{sizeM.ID, colorRed.ID}); err != nil {
		t.Fatalf("create variant B: %v", err)
	}

	facets, err := productRepo.AttributeFacets(ctx, []uuid.UUID{category.ID}, nil)
	if err != nil {
		t.Fatalf("attribute facets: %v", err)
	}
	facetValues := map[string][]string{}
	for _, f := range facets {
		for _, v := range f.Values {
			facetValues[f.AttributeName] = append(facetValues[f.AttributeName], v.Value)
		}
	}
	if len(facetValues[sizeAttr.Name]) != 2 || len(facetValues[colorAttr.Name]) != 2 {
		t.Fatalf("expected 2 size values and 2 color values in facets, got %+v", facetValues)
	}

	// AND across groups: Size=S AND Color=Blue matches only product A.
	andIDs, err := productRepo.ProductIDsByAttributeValues(ctx, []uuid.UUID{sizeS.ID, colorBlue.ID})
	if err != nil {
		t.Fatalf("product ids by attribute values (AND): %v", err)
	}
	if len(andIDs) != 1 || andIDs[0] != productA.ID {
		t.Fatalf("expected only product A for Size=S AND Color=Blue, got %v", andIDs)
	}

	// OR within group: Size=S OR Size=M matches both products.
	orIDs, err := productRepo.ProductIDsByAttributeValues(ctx, []uuid.UUID{sizeS.ID, sizeM.ID})
	if err != nil {
		t.Fatalf("product ids by attribute values (OR): %v", err)
	}
	if len(orIDs) != 2 {
		t.Fatalf("expected both products for Size=S OR Size=M, got %v", orIDs)
	}
}
