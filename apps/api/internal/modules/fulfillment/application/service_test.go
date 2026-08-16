package application_test

import (
	"context"
	"log/slog"
	"strings"
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
	lastCreateReq    application.CreateShipmentRequest
	createResult     application.ShipmentResult
	lastTrackBatch   []string
	trackResult      []application.TrackedParcel
	searchCalls      int
	lastSiteID       int64
	lastCalculateReq application.CalculateRequest
}

func (c *stubSpeedyClient) CreateShipment(_ context.Context, req application.CreateShipmentRequest) (application.ShipmentResult, error) {
	c.lastCreateReq = req
	return c.createResult, nil
}

func (c *stubSpeedyClient) Track(_ context.Context, _ application.Credentials, parcelIDs []string) ([]application.TrackedParcel, error) {
	c.lastTrackBatch = parcelIDs
	return c.trackResult, nil
}

func (c *stubSpeedyClient) Calculate(_ context.Context, req application.CalculateRequest) ([]application.CalculationResult, error) {
	c.lastCalculateReq = req
	results := make([]application.CalculationResult, 0, len(req.ServiceIDs))
	for _, id := range req.ServiceIDs {
		results = append(results, application.CalculationResult{
			ServiceID: id,
			Amount:    money.Money{AmountMinor: 500 + id, Currency: "EUR"},
		})
	}
	return results, nil
}

func (c *stubSpeedyClient) SearchOffices(_ context.Context, _ application.Credentials, siteID int64, name, officeType string) ([]application.Office, error) {
	c.searchCalls++
	c.lastSiteID = siteID
	return []application.Office{{ID: "o1", Name: name + " " + officeType, Type: officeType}}, nil
}

func (c *stubSpeedyClient) SearchSites(_ context.Context, _ application.Credentials, name string) ([]application.Site, error) {
	c.searchCalls++
	return []application.Site{{ID: 68134, Name: name}}, nil
}

func (c *stubSpeedyClient) SearchComplexes(_ context.Context, _ application.Credentials, siteID int64, name string) ([]application.Complex, error) {
	c.searchCalls++
	c.lastSiteID = siteID
	return []application.Complex{{ID: 1, Name: name}}, nil
}

func (c *stubSpeedyClient) SearchStreets(_ context.Context, _ application.Credentials, siteID int64, name string) ([]application.Street, error) {
	c.searchCalls++
	c.lastSiteID = siteID
	return []application.Street{{ID: 100, Name: name}}, nil
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

func TestSaveSettings_EmptyClearsAndAbsentPreserves(t *testing.T) {
	settings := &stubSettingsRepo{settings: map[string]domain.ProviderSettings{
		domain.ProviderSpeedy: {
			Provider: domain.ProviderSpeedy,
			Enabled:  true,
			Config: map[string]string{
				domain.SpeedyConfigUsername:                "api-user",
				domain.SpeedyConfigPassword:                "secret",
				domain.SpeedyConfigClientSystemID:          "117678825000",
				domain.SpeedyConfigDefaultCourierServiceID: "505",
			},
		},
	}}
	service := newTestService(settings, &stubSpeedyClient{}, &stubOrderGateway{})

	// Submit an explicit empty client system id (admin cleared the field). Password
	// is absent — the transport layer strips a blank secret before this point.
	saved, err := service.SaveSettings(context.Background(), domain.ProviderSpeedy, true, map[string]string{
		domain.SpeedyConfigUsername:                "api-user",
		domain.SpeedyConfigClientSystemID:          "",
		domain.SpeedyConfigDefaultCourierServiceID: "505",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := saved.Config[domain.SpeedyConfigClientSystemID]; got != "" {
		t.Errorf("expected client system id cleared, got %q", got)
	}
	if got := saved.Config[domain.SpeedyConfigPassword]; got != "secret" {
		t.Errorf("expected absent password preserved, got %q", got)
	}
	if got := saved.Config[domain.SpeedyConfigDefaultCourierServiceID]; got != "505" {
		t.Errorf("expected courier service id retained, got %q", got)
	}
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

func TestCreateShipmentForOrder_EasyBoxFallsBackToCourierServiceID(t *testing.T) {
	// EasyBox with no locker-specific service configured must fall back to the
	// courier service id (mirroring speedy_office), not send a blank service id.
	settings := &stubSettingsRepo{settings: map[string]domain.ProviderSettings{
		domain.ProviderSpeedy: {
			Provider: domain.ProviderSpeedy,
			Enabled:  true,
			Config: map[string]string{
				domain.SpeedyConfigUsername:                "api-user",
				domain.SpeedyConfigPassword:                "secret",
				domain.SpeedyConfigDefaultCourierServiceID: "505",
			},
		},
	}}
	speedy := &stubSpeedyClient{createResult: application.ShipmentResult{ShipmentID: "ship-1", ParcelID: "parcel-1"}}
	service := newTestService(settings, speedy, &stubOrderGateway{})

	if _, err := service.CreateShipmentForOrder(context.Background(), application.CreateShipmentInput{
		Provider:       domain.ProviderSpeedy,
		DeliveryMethod: "easybox",
		OfficeID:       "apt-7",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if speedy.lastCreateReq.ServiceID != "505" {
		t.Errorf("expected easybox to fall back to courier service id 505, got %q", speedy.lastCreateReq.ServiceID)
	}
}

func TestCreateShipmentForOrder_ContentsPreferInputAndTruncate(t *testing.T) {
	settings := &stubSettingsRepo{settings: map[string]domain.ProviderSettings{
		domain.ProviderSpeedy: {Provider: domain.ProviderSpeedy, Enabled: true, Config: map[string]string{
			domain.SpeedyConfigDefaultCourierServiceID: "505",
		}},
	}}
	speedy := &stubSpeedyClient{createResult: application.ShipmentResult{ShipmentID: "s", ParcelID: "p"}}
	service := newTestService(settings, speedy, &stubOrderGateway{})

	// 160 runes of multi-byte Cyrillic — must cap to 100 runes without splitting a
	// character, and mark the truncation with an ellipsis.
	long := strings.Repeat("Дъ", 80)
	if _, err := service.CreateShipmentForOrder(context.Background(), application.CreateShipmentInput{
		Provider: domain.ProviderSpeedy, DeliveryMethod: "speedy", Contents: long,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := []rune(speedy.lastCreateReq.Contents)
	if len(got) != 100 {
		t.Errorf("contents should be capped to 100 runes, got %d (%q)", len(got), speedy.lastCreateReq.Contents)
	}
	if got[len(got)-1] != '…' {
		t.Errorf("truncated contents should end with an ellipsis, got %q", string(got))
	}
}

func TestCreateShipmentForOrder_ContentsFallsBackToDefault(t *testing.T) {
	settings := &stubSettingsRepo{settings: map[string]domain.ProviderSettings{
		domain.ProviderSpeedy: {Provider: domain.ProviderSpeedy, Enabled: true, Config: map[string]string{
			domain.SpeedyConfigDefaultCourierServiceID: "505",
		}},
	}}
	speedy := &stubSpeedyClient{}
	service := newTestService(settings, speedy, &stubOrderGateway{})

	if _, err := service.CreateShipmentForOrder(context.Background(), application.CreateShipmentInput{
		Provider: domain.ProviderSpeedy, DeliveryMethod: "speedy", // no Contents
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if speedy.lastCreateReq.Contents != domain.DefaultParcelContents {
		t.Errorf("expected default contents %q, got %q", domain.DefaultParcelContents, speedy.lastCreateReq.Contents)
	}
	if speedy.lastCreateReq.Package != domain.DefaultParcelPackage {
		t.Errorf("expected default package %q, got %q", domain.DefaultParcelPackage, speedy.lastCreateReq.Package)
	}
}

func TestCreateShipmentForOrder_UsesRealWeightOverConfigDefault(t *testing.T) {
	settings := &stubSettingsRepo{settings: map[string]domain.ProviderSettings{
		domain.ProviderSpeedy: {
			Provider: domain.ProviderSpeedy,
			Enabled:  true,
			Config: map[string]string{
				domain.SpeedyConfigDefaultCourierServiceID: "505",
				domain.SpeedyConfigDefaultParcelWeightKg:   "2.5",
			},
		},
	}}
	speedy := &stubSpeedyClient{}
	service := newTestService(settings, speedy, &stubOrderGateway{})

	// A real per-order weight overrides the configured default...
	if _, err := service.CreateShipmentForOrder(context.Background(), application.CreateShipmentInput{
		Provider: domain.ProviderSpeedy, DeliveryMethod: "speedy", WeightKg: 0.8,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if speedy.lastCreateReq.ParcelWeightKg != 0.8 {
		t.Errorf("expected real weight 0.8, got %v", speedy.lastCreateReq.ParcelWeightKg)
	}

	// ...but a zero weight falls back to the configured default.
	if _, err := service.CreateShipmentForOrder(context.Background(), application.CreateShipmentInput{
		Provider: domain.ProviderSpeedy, DeliveryMethod: "speedy", WeightKg: 0,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if speedy.lastCreateReq.ParcelWeightKg != 2.5 {
		t.Errorf("expected fallback weight 2.5, got %v", speedy.lastCreateReq.ParcelWeightKg)
	}
}

func TestCalculateShippingCosts_MapsResultsToMethodsWithOfficeFallback(t *testing.T) {
	settings := &stubSettingsRepo{settings: map[string]domain.ProviderSettings{
		domain.ProviderSpeedy: {
			Provider: domain.ProviderSpeedy,
			Enabled:  true,
			Config: map[string]string{
				domain.SpeedyConfigUsername:                "api-user",
				domain.SpeedyConfigPassword:                "secret",
				domain.SpeedyConfigDefaultCourierServiceID: "505",
				// office service id left unset → falls back to the courier service.
				domain.SpeedyConfigDefaultLockerServiceID: "508",
			},
		},
	}}
	speedy := &stubSpeedyClient{}
	service := newTestService(settings, speedy, &stubOrderGateway{})

	quotes, err := service.CalculateShippingCosts(context.Background(), domain.ProviderSpeedy, 68134, 1.2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the two distinct service ids (courier 505, locker 508) are priced,
	// with the real weight passed through.
	if len(speedy.lastCalculateReq.ServiceIDs) != 2 {
		t.Fatalf("expected 2 distinct service ids, got %v", speedy.lastCalculateReq.ServiceIDs)
	}
	if speedy.lastCalculateReq.WeightKg != 1.2 {
		t.Errorf("expected weight 1.2 passed through, got %v", speedy.lastCalculateReq.WeightKg)
	}
	if speedy.lastCalculateReq.SiteID != 68134 {
		t.Errorf("expected site 68134, got %v", speedy.lastCalculateReq.SiteID)
	}

	// The courier service backs both door and office methods (office fell back),
	// so we get three quotes: speedy, speedy_office (both service 505) and easybox (508).
	byMethod := map[string]money.Money{}
	for _, q := range quotes {
		byMethod[q.DeliveryMethod] = q.Amount
	}
	if len(quotes) != 3 {
		t.Fatalf("expected 3 quotes, got %d (%+v)", len(quotes), quotes)
	}
	if byMethod["speedy"].AmountMinor != 505+500 {
		t.Errorf("expected speedy quote from service 505, got %+v", byMethod["speedy"])
	}
	if byMethod["speedy_office"].AmountMinor != 505+500 {
		t.Errorf("expected speedy_office quote from fallback service 505, got %+v", byMethod["speedy_office"])
	}
	if byMethod["easybox"].AmountMinor != 508+500 {
		t.Errorf("expected easybox quote from service 508, got %+v", byMethod["easybox"])
	}
}

func TestCalculateShippingCosts_ZeroWeightYieldsNoQuotes(t *testing.T) {
	settings := &stubSettingsRepo{settings: map[string]domain.ProviderSettings{
		domain.ProviderSpeedy: {
			Provider: domain.ProviderSpeedy,
			Enabled:  true,
			Config:   map[string]string{domain.SpeedyConfigDefaultCourierServiceID: "505"},
		},
	}}
	speedy := &stubSpeedyClient{}
	service := newTestService(settings, speedy, &stubOrderGateway{})

	quotes, err := service.CalculateShippingCosts(context.Background(), domain.ProviderSpeedy, 68134, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(quotes) != 0 {
		t.Fatalf("expected no quotes for zero weight, got %+v", quotes)
	}
	if len(speedy.lastCalculateReq.ServiceIDs) != 0 {
		t.Errorf("expected Calculate not to be called for zero weight")
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
