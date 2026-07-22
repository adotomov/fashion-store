package application

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/fulfillment/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/platform/metrics"
)

type Service struct {
	settings SettingsRepository
	speedy   SpeedyClient
	orders   OrderGateway
	notifier Notifier
	logger   *slog.Logger
}

func NewService(settings SettingsRepository, speedy SpeedyClient, orders OrderGateway, logger *slog.Logger) *Service {
	return &Service{settings: settings, speedy: speedy, orders: orders, logger: logger}
}

// WithNotifier attaches the customer-email notifier. Optional and chainable so
// tracking still works where email isn't configured.
func (s *Service) WithNotifier(notifier Notifier) *Service {
	s.notifier = notifier
	return s
}

func (s *Service) ListSettings(ctx context.Context) ([]domain.ProviderSettings, error) {
	return s.settings.List(ctx)
}

func (s *Service) GetSettings(ctx context.Context, provider string) (*domain.ProviderSettings, error) {
	return s.settings.Get(ctx, provider)
}

// SaveSettings merges the submitted config into whatever is already stored,
// skipping blank values — this is what lets the admin form re-save a
// provider (e.g. just flipping the toggle) without having to retype a
// password that's displayed masked.
func (s *Service) SaveSettings(ctx context.Context, provider string, enabled bool, config map[string]string) (*domain.ProviderSettings, error) {
	current, err := s.settings.Get(ctx, provider)
	if err != nil {
		return nil, err
	}

	merged := map[string]string{}
	if current != nil {
		for k, v := range current.Config {
			merged[k] = v
		}
	}
	for k, v := range config {
		if v == "" {
			continue
		}
		merged[k] = v
	}

	return s.settings.Save(ctx, domain.ProviderSettings{Provider: provider, Enabled: enabled, Config: merged})
}

func (s *Service) IsEnabled(ctx context.Context, provider string) bool {
	st, err := s.settings.Get(ctx, provider)
	if err != nil || st == nil {
		return false
	}
	return st.Enabled
}

func (s *Service) SearchOffices(ctx context.Context, provider, city, officeType string) ([]Office, error) {
	st, err := s.settings.Get(ctx, provider)
	if err != nil {
		return nil, err
	}
	if st == nil || !st.Enabled {
		return nil, domain.ErrProviderDisabled
	}
	return s.speedy.SearchOffices(ctx, credsFromConfig(st.Config), city, officeType)
}

// CreateShipmentForOrder builds and submits a Speedy shipment for a
// just-placed order. Callers should treat failures as non-fatal to checkout
// — the order stays placed without a tracking number, and an admin can
// retry fulfillment later.
func (s *Service) CreateShipmentForOrder(ctx context.Context, input CreateShipmentInput) (ShipmentResult, error) {
	st, err := s.settings.Get(ctx, input.Provider)
	if err != nil {
		return ShipmentResult{}, err
	}
	if st == nil || !st.Enabled {
		return ShipmentResult{}, domain.ErrProviderDisabled
	}

	serviceID := st.Config[domain.SpeedyConfigDefaultCourierServiceID]
	if input.DeliveryMethod == "easybox" {
		serviceID = st.Config[domain.SpeedyConfigDefaultLockerServiceID]
	}
	weight := parseFloatDefault(st.Config[domain.SpeedyConfigDefaultParcelWeightKg], 1.0)

	return s.speedy.CreateShipment(ctx, CreateShipmentRequest{
		Creds:          credsFromConfig(st.Config),
		ServiceID:      serviceID,
		ParcelWeightKg: weight,
		Recipient: ShipmentRecipient{
			ContactName: input.ContactName,
			Phone:       input.Phone,
			Email:       input.Email,
			City:        input.City,
			PostalCode:  input.PostalCode,
			Line1:       input.Line1,
			Line2:       input.Line2,
			CountryCode: input.CountryCode,
			OfficeID:    input.OfficeID,
		},
		CODAmount:  input.CODAmount,
		RequireCOD: input.RequireCOD,
		Ref1:       input.Ref1,
	})
}

// PollPendingShipments asks Speedy for the latest tracking operation of
// every order that has a Speedy shipment and isn't delivered/cancelled yet,
// batching parcel IDs to respect the documented 10-per-request limit.
func (s *Service) PollPendingShipments(ctx context.Context) error {
	refs, err := s.orders.ListAwaitingTracking(ctx)
	if err != nil {
		return err
	}
	if len(refs) == 0 {
		return nil
	}

	st, err := s.settings.Get(ctx, domain.ProviderSpeedy)
	if err != nil {
		return err
	}
	if st == nil || !st.Enabled {
		return nil
	}
	creds := credsFromConfig(st.Config)

	orderByParcel := make(map[string]TrackedOrderRef, len(refs))
	parcelIDs := make([]string, 0, len(refs))
	for _, ref := range refs {
		orderByParcel[ref.ParcelID] = ref
		parcelIDs = append(parcelIDs, ref.ParcelID)
	}

	const batchSize = 10
	for i := 0; i < len(parcelIDs); i += batchSize {
		batch := parcelIDs[i:min(i+batchSize, len(parcelIDs))]
		tracked, err := s.speedy.Track(ctx, creds, batch)
		if err != nil {
			s.logger.Error("speedy track request failed", "error", err)
			continue
		}
		for _, t := range tracked {
			ref, ok := orderByParcel[t.ParcelID]
			if !ok {
				continue
			}
			status := domain.FriendlyStatus(t.OperationCode, t.Description)
			update := ShipmentInfoUpdate{ShipmentStatus: &status}
			switch {
			case status == domain.StatusDelivered:
				delivered := "delivered"
				update.OrderStatus = &delivered
			case domain.IsInFlight(status):
				shipped := "shipped"
				update.OrderStatus = &shipped
			}
			if err := s.orders.SetShipmentInfo(ctx, ref.OrderID, update); err != nil {
				s.logger.Error("failed to update order shipment status", "error", err, "order_id", ref.OrderID)
			}
			// Tell the customer once, when the parcel first enters the carrier
			// network. The poll re-sees the same in-flight status every tick, so
			// the notifier's per-order dedupe key is what makes this send once.
			if domain.IsInFlight(status) {
				s.notifyShipped(ctx, ref)
			}
		}
	}
	return nil
}

// notifyShipped queues the dispatch notice. Tracking updates must never fail
// over an email, so problems are logged and swallowed.
func (s *Service) notifyShipped(ctx context.Context, ref TrackedOrderRef) {
	if s.notifier == nil || ref.ContactEmail == "" {
		return
	}
	notification := ShipmentNotification{
		OrderID:        ref.OrderID,
		OrderNumber:    ref.OrderNumber,
		CustomerName:   ref.ContactName,
		CustomerEmail:  ref.ContactEmail,
		Carrier:        ref.Carrier,
		TrackingNumber: ref.ParcelID,
	}
	if err := s.notifier.OrderShipped(ctx, notification); err != nil {
		s.logger.ErrorContext(ctx, "failed to queue shipping update email",
			"error", err, "order_id", ref.OrderID)
	}
}

// Run polls on a fixed interval until ctx is cancelled — intended to be
// launched as a single background goroutine for the lifetime of the server.
func (s *Service) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.PollPendingShipments(ctx); err != nil {
				metrics.FulfillmentPollError(ctx)
				s.logger.ErrorContext(ctx, "poll pending shipments failed", "error", err)
			}
		}
	}
}

func credsFromConfig(config map[string]string) Credentials {
	return Credentials{
		Username:       config[domain.SpeedyConfigUsername],
		Password:       config[domain.SpeedyConfigPassword],
		Language:       config[domain.SpeedyConfigLanguage],
		ClientSystemID: config[domain.SpeedyConfigClientSystemID],
	}
}

func parseFloatDefault(raw string, fallback float64) float64 {
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return v
}
