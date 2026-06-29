package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/inventory/domain"
)

// ReserveLine identifies a variant and quantity to hold during checkout.
// Variants with no inventory_items row (untracked) are silently skipped —
// untracked means "no known limit," matching the cart module's semantics.
type ReserveLine struct {
	VariantID uuid.UUID
	Quantity  int
}

type Repository interface {
	CreateItem(ctx context.Context, item domain.InventoryItem) (*domain.InventoryItem, error)
	ListItems(ctx context.Context) ([]domain.InventoryItem, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.InventoryItem, error)
	FindByVariantID(ctx context.Context, variantID uuid.UUID) (*domain.InventoryItem, error)
	UpdateSKU(ctx context.Context, id uuid.UUID, sku string) (*domain.InventoryItem, error)

	// AdjustStock atomically inserts a movement row and updates the item's
	// quantity_on_hand by quantityDelta in a single transaction.
	AdjustStock(ctx context.Context, itemID uuid.UUID, movementType domain.MovementType, quantityDelta int, note string, createdBy *uuid.UUID) (*domain.InventoryItem, *domain.InventoryMovement, error)

	ListMovements(ctx context.Context, itemID uuid.UUID) ([]domain.InventoryMovement, error)

	// ReserveForVariants holds stock for each tracked line (raising
	// quantity_reserved) inside one transaction, failing the whole
	// reservation if any tracked line doesn't have enough available stock.
	ReserveForVariants(ctx context.Context, lines []ReserveLine, createdBy *uuid.UUID) (*domain.Reservation, error)
	// CommitReservation permanently consumes the held stock (decrementing
	// quantity_on_hand and quantity_reserved together) once an order is
	// actually placed.
	CommitReservation(ctx context.Context, reservationID uuid.UUID, createdBy *uuid.UUID) error
	// ReleaseReservation gives back the held quantity_reserved without
	// touching quantity_on_hand, e.g. when a card_online charge fails.
	ReleaseReservation(ctx context.Context, reservationID uuid.UUID, createdBy *uuid.UUID) error
}
