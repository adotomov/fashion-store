package infrastructure_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	catalogdomain "github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
	cataloginfra "github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/infrastructure"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/inventory/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/inventory/infrastructure"
)

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

func TestPostgresRepository_FullLifecycle(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	productRepo := cataloginfra.NewPostgresProductRepository(pool)
	inventoryRepo := infrastructure.NewPostgresRepository(pool)

	product, err := productRepo.Create(ctx, catalogdomain.Product{
		Name: "IT-Test Inventory Product", Slug: "it-test-inventory-product", Status: catalogdomain.ProductStatusDraft,
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	t.Cleanup(func() { _ = productRepo.Delete(ctx, product.ID) })

	variant, err := productRepo.CreateVariant(ctx, catalogdomain.ProductVariant{ProductID: product.ID}, nil)
	if err != nil {
		t.Fatalf("create variant: %v", err)
	}

	item, err := inventoryRepo.CreateItem(ctx, domain.InventoryItem{VariantID: variant.ID, SKU: "IT-TEST-SKU-001"})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	// Duplicate SKU should be rejected.
	if _, err := inventoryRepo.CreateItem(ctx, domain.InventoryItem{VariantID: variant.ID, SKU: "IT-TEST-SKU-001"}); err == nil {
		t.Fatal("expected error creating item with duplicate variant/sku")
	}

	updatedItem, movement, err := inventoryRepo.AdjustStock(ctx, item.ID, domain.MovementInitialStock, 20, "initial stock", nil)
	if err != nil {
		t.Fatalf("adjust stock: %v", err)
	}
	if updatedItem.QuantityOnHand != 20 {
		t.Errorf("expected quantity on hand 20, got %d", updatedItem.QuantityOnHand)
	}
	if movement.Type != domain.MovementInitialStock || movement.QuantityDelta != 20 {
		t.Errorf("unexpected movement: %+v", movement)
	}

	if _, _, err := inventoryRepo.AdjustStock(ctx, item.ID, domain.MovementAdminAdjustment, -100, "too much", nil); err == nil {
		t.Fatal("expected error for resulting negative stock")
	}

	items, err := inventoryRepo.ListItems(ctx)
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	found := false
	for _, i := range items {
		if i.ID == item.ID {
			found = true
			if i.ProductName != "IT-Test Inventory Product" {
				t.Errorf("expected joined product name, got %q", i.ProductName)
			}
		}
	}
	if !found {
		t.Error("expected created item to appear in ListItems")
	}

	loaded, err := inventoryRepo.FindByID(ctx, item.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if loaded.ProductName != "IT-Test Inventory Product" {
		t.Errorf("expected joined product name on FindByID, got %q", loaded.ProductName)
	}

	movements, err := inventoryRepo.ListMovements(ctx, item.ID)
	if err != nil {
		t.Fatalf("list movements: %v", err)
	}
	if len(movements) != 1 {
		t.Errorf("expected 1 movement, got %d", len(movements))
	}
}
