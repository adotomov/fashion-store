package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/inventory/domain"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateItem assigns a SKU to a variant and, optionally, records an
// initial_stock movement in the same operation.
func (s *Service) CreateItem(ctx context.Context, variantID uuid.UUID, sku string, initialQuantity int, createdBy *uuid.UUID) (*domain.InventoryItem, error) {
	if sku == "" {
		return nil, domain.ValidationError("sku is required")
	}
	if initialQuantity < 0 {
		return nil, domain.ValidationError("initial quantity cannot be negative")
	}

	item, err := s.repo.CreateItem(ctx, domain.InventoryItem{VariantID: variantID, SKU: sku})
	if err != nil {
		return nil, err
	}

	if initialQuantity > 0 {
		item, _, err = s.repo.AdjustStock(ctx, item.ID, domain.MovementInitialStock, initialQuantity, "Initial stock", createdBy)
		if err != nil {
			return nil, err
		}
	}

	return item, nil
}

func (s *Service) ListItems(ctx context.Context) ([]domain.InventoryItem, error) {
	return s.repo.ListItems(ctx)
}

func (s *Service) GetItem(ctx context.Context, id uuid.UUID) (*domain.InventoryItem, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) UpdateSKU(ctx context.Context, id uuid.UUID, sku string) (*domain.InventoryItem, error) {
	if sku == "" {
		return nil, domain.ValidationError("sku is required")
	}
	return s.repo.UpdateSKU(ctx, id, sku)
}

// AdjustStock is the admin-facing stock change entry point. Only
// admin-adjustable movement types are accepted — reservation/sale_committed
// movements are reserved for the (future) checkout flow.
func (s *Service) AdjustStock(ctx context.Context, itemID uuid.UUID, movementType domain.MovementType, quantityDelta int, note string, createdBy *uuid.UUID) (*domain.InventoryItem, error) {
	if !movementType.AdminAdjustable() {
		return nil, domain.ErrInvalidMovementType
	}
	if quantityDelta == 0 {
		return nil, domain.ValidationError("quantity delta cannot be zero")
	}

	item, _, err := s.repo.AdjustStock(ctx, itemID, movementType, quantityDelta, note, createdBy)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Service) ListMovements(ctx context.Context, itemID uuid.UUID) ([]domain.InventoryMovement, error) {
	return s.repo.ListMovements(ctx, itemID)
}

// ReserveForVariants is called by the checkout flow before payment is
// processed, so stock is held while a card_online charge is in flight.
func (s *Service) ReserveForVariants(ctx context.Context, lines []ReserveLine, createdBy *uuid.UUID) (*domain.Reservation, error) {
	return s.repo.ReserveForVariants(ctx, lines, createdBy)
}

// CommitReservation finalizes a reservation once the order is actually
// placed (payment succeeded, or no upfront payment was required).
func (s *Service) CommitReservation(ctx context.Context, reservationID uuid.UUID, createdBy *uuid.UUID) error {
	return s.repo.CommitReservation(ctx, reservationID, createdBy)
}

// ReleaseReservation gives back held stock when checkout fails after
// reserving it, e.g. a declined card_online charge.
func (s *Service) ReleaseReservation(ctx context.Context, reservationID uuid.UUID, createdBy *uuid.UUID) error {
	return s.repo.ReleaseReservation(ctx, reservationID, createdBy)
}
