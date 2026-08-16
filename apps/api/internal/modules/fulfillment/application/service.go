package application

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/fulfillment/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/platform/metrics"
)

type Service struct {
	settings  SettingsRepository
	speedy    SpeedyClient
	orders    OrderGateway
	notifier  Notifier
	logger    *slog.Logger
	locations *locationCache
}

// locationCacheTTL is how long resolved Speedy location lookups are trusted.
// Location data changes on the order of days, so a long TTL is safe and keeps
// typeahead snappy.
const locationCacheTTL = 12 * time.Hour

func NewService(settings SettingsRepository, speedy SpeedyClient, orders OrderGateway, logger *slog.Logger) *Service {
	return &Service{
		settings:  settings,
		speedy:    speedy,
		orders:    orders,
		logger:    logger,
		locations: newLocationCache(locationCacheTTL),
	}
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
	// A submitted key overwrites the stored value, and an empty value clears it —
	// so the admin can blank out a field (e.g. an unwanted client system id). The
	// transport layer strips secret fields (password) before they reach here when
	// blank, so clearing never wipes a stored secret. Keys absent from the submit
	// (fields the form doesn't render) keep their stored value.
	for k, v := range config {
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

// enabledCreds resolves a provider's live credentials, rejecting a disabled or
// missing provider. Shared by every Speedy read path (office/location search).
func (s *Service) enabledCreds(ctx context.Context, provider string) (Credentials, error) {
	st, err := s.settings.Get(ctx, provider)
	if err != nil {
		return Credentials{}, err
	}
	if st == nil || !st.Enabled {
		return Credentials{}, domain.ErrProviderDisabled
	}
	return credsFromConfig(st.Config), nil
}

// SearchOffices lists Speedy offices ("OFFICE") or EasyBox lockers ("APT"),
// optionally scoped to a site and/or name fragment. Results are cached in
// memory keyed by type + site + query.
func (s *Service) SearchOffices(ctx context.Context, provider string, siteID int64, name, officeType string) ([]Office, error) {
	creds, err := s.enabledCreds(ctx, provider)
	if err != nil {
		return nil, err
	}
	key := locationCacheKey("office:"+officeType, siteID, name)
	return cachedLookup(s.locations, key, time.Now(), func() ([]Office, error) {
		return s.speedy.SearchOffices(ctx, creds, siteID, name, officeType)
	})
}

// SearchSites resolves populated places (cities/towns) by name fragment for
// the address typeahead.
func (s *Service) SearchSites(ctx context.Context, provider, query string) ([]Site, error) {
	creds, err := s.enabledCreds(ctx, provider)
	if err != nil {
		return nil, err
	}
	key := locationCacheKey("site", 0, query)
	return cachedLookup(s.locations, key, time.Now(), func() ([]Site, error) {
		return s.speedy.SearchSites(ctx, creds, query)
	})
}

// SearchComplexes lists residential complexes (кв./жк.) within a site.
func (s *Service) SearchComplexes(ctx context.Context, provider string, siteID int64, query string) ([]Complex, error) {
	creds, err := s.enabledCreds(ctx, provider)
	if err != nil {
		return nil, err
	}
	key := locationCacheKey("complex", siteID, query)
	return cachedLookup(s.locations, key, time.Now(), func() ([]Complex, error) {
		return s.speedy.SearchComplexes(ctx, creds, siteID, query)
	})
}

// SearchStreets lists streets within a site.
func (s *Service) SearchStreets(ctx context.Context, provider string, siteID int64, query string) ([]Street, error) {
	creds, err := s.enabledCreds(ctx, provider)
	if err != nil {
		return nil, err
	}
	key := locationCacheKey("street", siteID, query)
	return cachedLookup(s.locations, key, time.Now(), func() ([]Street, error) {
		return s.speedy.SearchStreets(ctx, creds, siteID, query)
	})
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

	// Pick the Speedy service ID for the chosen delivery shape. Office and
	// locker each have their own configured service; both fall back to the
	// courier service if unset (office pickup is often the same courier service
	// with an office destination).
	serviceID := st.Config[domain.SpeedyConfigDefaultCourierServiceID]
	switch input.DeliveryMethod {
	case "speedy_office":
		if office := st.Config[domain.SpeedyConfigDefaultOfficeServiceID]; office != "" {
			serviceID = office
		}
	case "easybox":
		if locker := st.Config[domain.SpeedyConfigDefaultLockerServiceID]; locker != "" {
			serviceID = locker
		}
	}
	// Use the real per-order parcel weight; fall back to the provider's
	// configured default when the order carried no SKU weights.
	weight := input.WeightKg
	if weight <= 0 {
		weight = parseFloatDefault(st.Config[domain.SpeedyConfigDefaultParcelWeightKg], 1.0)
	}

	// Speedy requires a content description and package type on every shipment.
	// Prefer the per-order description derived from the item names, then the
	// configured store default, then a constant — and cap to Speedy's field
	// limits (contents 100, package 50) with a rune-safe truncation.
	contents := truncateRunes(firstNonEmpty(input.Contents,
		firstNonEmpty(st.Config[domain.SpeedyConfigDefaultParcelContents], domain.DefaultParcelContents)), 100)
	pkg := truncateRunes(firstNonEmpty(st.Config[domain.SpeedyConfigDefaultParcelPackage], domain.DefaultParcelPackage), 50)

	return s.speedy.CreateShipment(ctx, CreateShipmentRequest{
		Creds:          credsFromConfig(st.Config),
		ServiceID:      serviceID,
		ParcelWeightKg: weight,
		Contents:       contents,
		Package:        pkg,
		Recipient: ShipmentRecipient{
			ContactName: input.ContactName,
			Phone:       input.Phone,
			Email:       input.Email,
			CountryID:   input.CountryID,
			SiteID:      input.SiteID,
			ComplexID:   input.ComplexID,
			StreetID:    input.StreetID,
			StreetNo:    input.StreetNo,
			BlockNo:     input.BlockNo,
			EntranceNo:  input.EntranceNo,
			FloorNo:     input.FloorNo,
			ApartmentNo: input.ApartmentNo,
			OfficeID:    input.OfficeID,
		},
		CODAmount:  input.CODAmount,
		RequireCOD: input.RequireCOD,
		Ref1:       input.Ref1,
	})
}

// deliveryMethod codes as used by the checkout module. Duplicated as string
// literals here (rather than importing checkout) to keep the module boundary —
// the same approach CreateShipmentForOrder already takes.
const (
	deliveryMethodSpeedy       = "speedy"
	deliveryMethodSpeedyOffice = "speedy_office"
	deliveryMethodEasyBox      = "easybox"
)

// CalculateShippingCosts prices every configured Speedy delivery service for a
// parcel of weightKg to the given destination site, in a single /calculate
// call, and returns one quote per delivery method that Speedy could price. Used
// to record what shipping would cost per order (never shown to anyone). A zero
// weight or no configured services yields no quotes.
func (s *Service) CalculateShippingCosts(ctx context.Context, provider string, siteID int64, weightKg float64) ([]ShippingCostQuote, error) {
	st, err := s.settings.Get(ctx, provider)
	if err != nil {
		return nil, err
	}
	if st == nil || !st.Enabled {
		return nil, domain.ErrProviderDisabled
	}

	// Map each configured service id to the delivery method(s) it fulfils. A
	// single service can back several methods (office falls back to courier).
	courier := st.Config[domain.SpeedyConfigDefaultCourierServiceID]
	office := st.Config[domain.SpeedyConfigDefaultOfficeServiceID]
	if office == "" {
		office = courier
	}
	locker := st.Config[domain.SpeedyConfigDefaultLockerServiceID]

	methodsByService := map[string][]string{}
	addService := func(serviceID, method string) {
		if serviceID != "" {
			methodsByService[serviceID] = append(methodsByService[serviceID], method)
		}
	}
	addService(courier, deliveryMethodSpeedy)
	addService(office, deliveryMethodSpeedyOffice)
	addService(locker, deliveryMethodEasyBox)

	serviceIDs := make([]int64, 0, len(methodsByService))
	for id := range methodsByService {
		n, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			continue
		}
		serviceIDs = append(serviceIDs, n)
	}
	if len(serviceIDs) == 0 || weightKg <= 0 {
		return nil, nil
	}

	results, err := s.speedy.Calculate(ctx, CalculateRequest{
		Creds:      credsFromConfig(st.Config),
		SiteID:     siteID,
		WeightKg:   weightKg,
		ServiceIDs: serviceIDs,
	})
	if err != nil {
		return nil, err
	}

	quotes := make([]ShippingCostQuote, 0, len(results))
	for _, r := range results {
		idStr := strconv.FormatInt(r.ServiceID, 10)
		for _, method := range methodsByService[idStr] {
			quotes = append(quotes, ShippingCostQuote{DeliveryMethod: method, ServiceID: idStr, Amount: r.Amount})
		}
	}
	return quotes, nil
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
		Locale:         ref.Locale,
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

// firstNonEmpty returns the trimmed value if non-blank, else the fallback —
// used to back the mandatory Speedy content fields with sensible defaults.
func firstNonEmpty(value, fallback string) string {
	if v := strings.TrimSpace(value); v != "" {
		return v
	}
	return fallback
}

// truncateRunes caps s to max runes (not bytes, so multi-byte Cyrillic never
// splits mid-character), replacing the tail with an ellipsis when it overflows —
// so a long item-derived description still fits Speedy's fixed field limits.
func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
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
