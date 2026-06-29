package application_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/inventory/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/inventory/domain"
)

type fakeRepo struct {
	byID      map[uuid.UUID]domain.InventoryItem
	byVariant map[uuid.UUID]uuid.UUID
	skusUsed  map[string]bool
	movements map[uuid.UUID][]domain.InventoryMovement
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		byID:      map[uuid.UUID]domain.InventoryItem{},
		byVariant: map[uuid.UUID]uuid.UUID{},
		skusUsed:  map[string]bool{},
		movements: map[uuid.UUID][]domain.InventoryMovement{},
	}
}

func (f *fakeRepo) CreateItem(_ context.Context, item domain.InventoryItem) (*domain.InventoryItem, error) {
	if f.skusUsed[item.SKU] {
		return nil, domain.ErrSKUConflict
	}
	if _, ok := f.byVariant[item.VariantID]; ok {
		return nil, domain.ErrItemAlreadyExists
	}
	item.ID = uuid.New()
	f.skusUsed[item.SKU] = true
	f.byVariant[item.VariantID] = item.ID
	f.byID[item.ID] = item
	return &item, nil
}

func (f *fakeRepo) ListItems(_ context.Context) ([]domain.InventoryItem, error) {
	var out []domain.InventoryItem
	for _, i := range f.byID {
		out = append(out, i)
	}
	return out, nil
}

func (f *fakeRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.InventoryItem, error) {
	i, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrItemNotFound
	}
	return &i, nil
}

func (f *fakeRepo) FindByVariantID(_ context.Context, variantID uuid.UUID) (*domain.InventoryItem, error) {
	id, ok := f.byVariant[variantID]
	if !ok {
		return nil, domain.ErrItemNotFound
	}
	i := f.byID[id]
	return &i, nil
}

func (f *fakeRepo) UpdateSKU(_ context.Context, id uuid.UUID, sku string) (*domain.InventoryItem, error) {
	i, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrItemNotFound
	}
	if f.skusUsed[sku] {
		return nil, domain.ErrSKUConflict
	}
	i.SKU = sku
	f.byID[id] = i
	return &i, nil
}

func (f *fakeRepo) AdjustStock(_ context.Context, itemID uuid.UUID, movementType domain.MovementType, quantityDelta int, note string, createdBy *uuid.UUID) (*domain.InventoryItem, *domain.InventoryMovement, error) {
	i, ok := f.byID[itemID]
	if !ok {
		return nil, nil, domain.ErrItemNotFound
	}
	if i.QuantityOnHand+quantityDelta < 0 {
		return nil, nil, domain.ErrInsufficientStock
	}
	i.QuantityOnHand += quantityDelta
	f.byID[itemID] = i

	m := domain.InventoryMovement{ID: uuid.New(), InventoryItemID: itemID, Type: movementType, QuantityDelta: quantityDelta, Note: note, CreatedBy: createdBy}
	f.movements[itemID] = append(f.movements[itemID], m)

	return &i, &m, nil
}

func (f *fakeRepo) ListMovements(_ context.Context, itemID uuid.UUID) ([]domain.InventoryMovement, error) {
	return f.movements[itemID], nil
}

func (f *fakeRepo) ReserveForVariants(_ context.Context, _ []application.ReserveLine, _ *uuid.UUID) (*domain.Reservation, error) {
	return nil, nil
}

func (f *fakeRepo) CommitReservation(_ context.Context, _ uuid.UUID, _ *uuid.UUID) error {
	return nil
}

func (f *fakeRepo) ReleaseReservation(_ context.Context, _ uuid.UUID, _ *uuid.UUID) error {
	return nil
}

func TestCreateItem_WithInitialQuantityRecordsMovement(t *testing.T) {
	svc := application.NewService(newFakeRepo())
	variantID := uuid.New()

	item, err := svc.CreateItem(context.Background(), variantID, "SKU-001", 10, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.QuantityOnHand != 10 {
		t.Errorf("expected quantity on hand 10, got %d", item.QuantityOnHand)
	}

	movements, err := svc.ListMovements(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(movements) != 1 || movements[0].Type != domain.MovementInitialStock {
		t.Fatalf("expected one initial_stock movement, got %+v", movements)
	}
}

func TestCreateItem_RejectsEmptySKU(t *testing.T) {
	svc := application.NewService(newFakeRepo())
	if _, err := svc.CreateItem(context.Background(), uuid.New(), "", 0, nil); err == nil {
		t.Fatal("expected error for empty sku")
	}
}

func TestAdjustStock_RejectsNonAdminAdjustableType(t *testing.T) {
	repo := newFakeRepo()
	svc := application.NewService(repo)

	item, err := svc.CreateItem(context.Background(), uuid.New(), "SKU-002", 5, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := svc.AdjustStock(context.Background(), item.ID, domain.MovementSaleCommitted, -1, "should be rejected", nil); err == nil {
		t.Fatal("expected error for non-admin-adjustable movement type")
	}
}

func TestAdjustStock_RejectsNegativeResultingStock(t *testing.T) {
	repo := newFakeRepo()
	svc := application.NewService(repo)

	item, err := svc.CreateItem(context.Background(), uuid.New(), "SKU-003", 2, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := svc.AdjustStock(context.Background(), item.ID, domain.MovementAdminAdjustment, -5, "too much", nil); err == nil {
		t.Fatal("expected error for resulting negative stock")
	}
}

func TestAdjustStock_AllowsAdminAdjustment(t *testing.T) {
	repo := newFakeRepo()
	svc := application.NewService(repo)

	item, err := svc.CreateItem(context.Background(), uuid.New(), "SKU-004", 5, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, err := svc.AdjustStock(context.Background(), item.ID, domain.MovementAdminAdjustment, 3, "restock", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.QuantityOnHand != 8 {
		t.Errorf("expected quantity on hand 8, got %d", updated.QuantityOnHand)
	}
}
