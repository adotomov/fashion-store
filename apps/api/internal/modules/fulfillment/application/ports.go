package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/fulfillment/domain"
)

// SettingsRepository persists per-provider configuration.
type SettingsRepository interface {
	Get(ctx context.Context, provider string) (*domain.ProviderSettings, error)
	Save(ctx context.Context, settings domain.ProviderSettings) (*domain.ProviderSettings, error)
	List(ctx context.Context) ([]domain.ProviderSettings, error)
}

// SpeedyClient is the real HTTP client against the Speedy Web API. Credentials
// are passed per-call (resolved from settings at call time) rather than
// baked into the client, since an admin can change or disable them at any
// time.
type SpeedyClient interface {
	CreateShipment(ctx context.Context, req CreateShipmentRequest) (ShipmentResult, error)
	Track(ctx context.Context, creds Credentials, parcelIDs []string) ([]TrackedParcel, error)
	SearchOffices(ctx context.Context, creds Credentials, city, officeType string) ([]Office, error)
}

// TrackedOrderRef is the minimal info the poller needs to ask Speedy for an
// update and write it back.
type TrackedOrderRef struct {
	OrderID  uuid.UUID
	ParcelID string
}

// ShipmentInfoUpdate mirrors the orders module's UpdateFulfillmentInput
// shape — all fields optional, only non-nil ones are changed.
type ShipmentInfoUpdate struct {
	Carrier        *string
	TrackingNumber *string
	ShipmentID     *string
	ShipmentStatus *string
	OrderStatus    *string
}

// OrderGateway lets fulfillment read/write order shipment state without
// importing the orders module's domain or repository directly.
type OrderGateway interface {
	ListAwaitingTracking(ctx context.Context) ([]TrackedOrderRef, error)
	SetShipmentInfo(ctx context.Context, orderID uuid.UUID, update ShipmentInfoUpdate) error
}
