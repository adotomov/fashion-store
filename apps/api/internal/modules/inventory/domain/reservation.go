package domain

import (
	"time"

	"github.com/google/uuid"
)

// Reservation and ReservationItem model short-TTL inventory holds during
// checkout: stock is reserved while a (possibly card_online) payment is
// processed, then either committed (decrementing on-hand stock for good)
// or released (giving the reserved quantity back) depending on the
// outcome. The checkout module drives this via Service.ReserveForVariants /
// CommitReservation / ReleaseReservation.
type ReservationStatus string

const (
	ReservationPending   ReservationStatus = "pending"
	ReservationCommitted ReservationStatus = "committed"
	ReservationReleased  ReservationStatus = "released"
	ReservationExpired   ReservationStatus = "expired"
)

type Reservation struct {
	ID        uuid.UUID
	Status    ReservationStatus
	ExpiresAt *time.Time
	Items     []ReservationItem
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ReservationItem struct {
	ID              uuid.UUID
	ReservationID   uuid.UUID
	InventoryItemID uuid.UUID
	Quantity        int
}
