package domain

import (
	"time"

	"github.com/google/uuid"
)

type MovementType string

const (
	MovementInitialStock       MovementType = "initial_stock"
	MovementAdminAdjustment    MovementType = "admin_adjustment"
	MovementReservation        MovementType = "reservation"
	MovementReservationRelease MovementType = "reservation_release"
	MovementSaleCommitted      MovementType = "sale_committed"
	MovementReturn             MovementType = "return"
	MovementManualCorrection   MovementType = "manual_correction"
)

func (t MovementType) Valid() bool {
	switch t {
	case MovementInitialStock, MovementAdminAdjustment, MovementReservation,
		MovementReservationRelease, MovementSaleCommitted, MovementReturn, MovementManualCorrection:
		return true
	default:
		return false
	}
}

// AdminAdjustable reports whether a human admin may directly trigger this
// movement type. Reservation/sale_committed movements are only ever
// produced by the (future) checkout flow, never by a manual admin action.
func (t MovementType) AdminAdjustable() bool {
	switch t {
	case MovementAdminAdjustment, MovementReturn, MovementManualCorrection:
		return true
	default:
		return false
	}
}

type InventoryMovement struct {
	ID              uuid.UUID
	InventoryItemID uuid.UUID
	Type            MovementType
	QuantityDelta   int
	Note            string
	CreatedBy       *uuid.UUID
	CreatedAt       time.Time
}
