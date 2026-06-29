package application_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/fulfillment/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/fulfillment/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

type stubSettingsRepo struct {
	settings map[string]domain.ProviderSettings
}

func (r *stubSettingsRepo) Get(_ context.Context, provider string) (*domain.ProviderSettings, error) {
	s, ok := r.settings[provider]
	if !ok {
		return nil, nil
	}
	return &s, nil
}

func (r *stubSettingsRepo) Save(_ context.Context, settings domain.ProviderSettings) (*domain.ProviderSettings, error) {
	r.settings[settings.Provider] = settings
	saved := settings
	return &saved, nil
}

func (r *stubSettingsRepo) List(_ context.Context) ([]domain.ProviderSettings, error) {
	result := make([]domain.ProviderSettings, 0, len(r.settings))
	for _, s := range r.settings {
		result = append(result, s)
	}
	return result, nil
}

type stubSpeedyClient struct {
	lastCreateReq  application.CreateShipmentRequest
	createResult   application.ShipmentResult
	lastTrackBatch []string
	trackResult    []application.TrackedParcel
}

func (c *stubSpeedyClient) CreateShipment(_ context.Context, req application.CreateShipmentRequest) (application.ShipmentResult, error) {
	c.lastCreateReq = req
	return c.createResult, nil
}

func (c *stubSpeedyClient) Track(_ context.Context, _ application.Credentials, parcelIDs []string) ([]application.TrackedParcel, error) {
	c.lastTrackBatch = parcelIDs
	return c.trackResult, nil
}

func (c *stubSpeedyClient) SearchOffices(context.Context, application.Credentials, string, string) ([]application.Office, error) {
	return nil, nil
}

type stubOrderGateway struct {
	refs       []application.TrackedOrderRef
	lastUpdate application.ShipmentInfoUpdate
	lastOrder  uuid.UUID
}

func (g *stubOrderGateway) ListAwaitingTracking(context.Context) ([]application.TrackedOrderRef, error) {
	return g.refs, nil
}

func (g *stubOrderGateway) SetShipmentInfo(_ context.Context, orderID uuid.UUID, update application.ShipmentInfoUpdate) error {
	g.lastOrder = orderID
	g.lastUpdate = update
	return nil
}

func newTestService(settings *stubSettingsRepo, speedy *stubSpeedyClient, orders *stubOrderGateway) *application.Service {
	return application.NewService(settings, speedy, orders, slog.Default())
}

func TestCreateShipmentForOrder_BuildsRequestFromSettings(t *testing.T) {
	settings := &stubSettingsRepo{settings: map[string]domain.ProviderSettings{
		domain.ProviderSpeedy: {
			Provider: domain.ProviderSpeedy,
			Enabled:  true,
			Config: map[string]string{
				domain.SpeedyConfigUsername:                "api-user",
				domain.SpeedyConfigPassword:                "secret",
				domain.SpeedyConfigDefaultCourierServiceID: "505",
				domain.SpeedyConfigDefaultLockerServiceID:  "508",
				domain.SpeedyConfigDefaultParcelWeightKg:   "2.5",
			},
		},
	}}
	speedy := &stubSpeedyClient{createResult: application.ShipmentResult{ShipmentID: "ship-1", ParcelID: "parcel-1"}}
	orders := &stubOrderGateway{}
	service := newTestService(settings, speedy, orders)

	result, err := service.CreateShipmentForOrder(context.Background(), application.CreateShipmentInput{
		Provider:       domain.ProviderSpeedy,
		DeliveryMethod: "easybox",
		ContactName:    "Jane Doe",
		Phone:          "0888123456",
		OfficeID:       "office-42",
		RequireCOD:     true,
		CODAmount:      money.Money{AmountMinor: 1999, Currency: "EUR"},
		Ref1:           "ORD-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ShipmentID != "ship-1" || result.ParcelID != "parcel-1" {
		t.Fatalf("unexpected result: %+v", result)
	}

	if speedy.lastCreateReq.ServiceID != "508" {
		t.Errorf("expected locker service id 508 for easybox, got %q", speedy.lastCreateReq.ServiceID)
	}
	if speedy.lastCreateReq.ParcelWeightKg != 2.5 {
		t.Errorf("expected configured weight 2.5, got %v", speedy.lastCreateReq.ParcelWeightKg)
	}
	if speedy.lastCreateReq.Recipient.OfficeID != "office-42" {
		t.Errorf("expected office id passed through, got %q", speedy.lastCreateReq.Recipient.OfficeID)
	}
	if !speedy.lastCreateReq.RequireCOD || speedy.lastCreateReq.CODAmount.AmountMinor != 1999 {
		t.Errorf("expected COD required with amount 1999, got %+v", speedy.lastCreateReq)
	}
}

func TestCreateShipmentForOrder_DisabledProvider(t *testing.T) {
	settings := &stubSettingsRepo{settings: map[string]domain.ProviderSettings{
		domain.ProviderSpeedy: {Provider: domain.ProviderSpeedy, Enabled: false},
	}}
	service := newTestService(settings, &stubSpeedyClient{}, &stubOrderGateway{})

	_, err := service.CreateShipmentForOrder(context.Background(), application.CreateShipmentInput{Provider: domain.ProviderSpeedy})
	if err != domain.ErrProviderDisabled {
		t.Fatalf("expected ErrProviderDisabled, got %v", err)
	}
}

func TestPollPendingShipments_MapsOperationCodeAndBumpsOrderStatus(t *testing.T) {
	orderID := uuid.New()
	settings := &stubSettingsRepo{settings: map[string]domain.ProviderSettings{
		domain.ProviderSpeedy: {Provider: domain.ProviderSpeedy, Enabled: true, Config: map[string]string{}},
	}}
	speedy := &stubSpeedyClient{trackResult: []application.TrackedParcel{
		{ParcelID: "parcel-1", OperationCode: 14, Description: "Delivered"},
	}}
	orders := &stubOrderGateway{refs: []application.TrackedOrderRef{{OrderID: orderID, ParcelID: "parcel-1"}}}
	service := newTestService(settings, speedy, orders)

	if err := service.PollPendingShipments(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if orders.lastOrder != orderID {
		t.Fatalf("expected update for order %s, got %s", orderID, orders.lastOrder)
	}
	if orders.lastUpdate.ShipmentStatus == nil || *orders.lastUpdate.ShipmentStatus != "delivered" {
		t.Fatalf("expected shipment status 'delivered', got %+v", orders.lastUpdate.ShipmentStatus)
	}
	if orders.lastUpdate.OrderStatus == nil || *orders.lastUpdate.OrderStatus != "delivered" {
		t.Fatalf("expected order status bumped to 'delivered', got %+v", orders.lastUpdate.OrderStatus)
	}
}

func TestPollPendingShipments_InFlightBumpsToShipped(t *testing.T) {
	orderID := uuid.New()
	settings := &stubSettingsRepo{settings: map[string]domain.ProviderSettings{
		domain.ProviderSpeedy: {Provider: domain.ProviderSpeedy, Enabled: true, Config: map[string]string{}},
	}}
	speedy := &stubSpeedyClient{trackResult: []application.TrackedParcel{
		{ParcelID: "parcel-1", OperationCode: 12, Description: "Out for Delivery"},
	}}
	orders := &stubOrderGateway{refs: []application.TrackedOrderRef{{OrderID: orderID, ParcelID: "parcel-1"}}}
	service := newTestService(settings, speedy, orders)

	if err := service.PollPendingShipments(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if orders.lastUpdate.ShipmentStatus == nil || *orders.lastUpdate.ShipmentStatus != "out_for_delivery" {
		t.Fatalf("expected shipment status 'out_for_delivery', got %+v", orders.lastUpdate.ShipmentStatus)
	}
	if orders.lastUpdate.OrderStatus == nil || *orders.lastUpdate.OrderStatus != "shipped" {
		t.Fatalf("expected order status bumped to 'shipped', got %+v", orders.lastUpdate.OrderStatus)
	}
}
