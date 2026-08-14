package infrastructure

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/fulfillment/application"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

// FakeSpeedyClient is a local stand-in for the real Speedy Web API, selected
// when SPEEDY_MODE=fake. It never makes a network call: shipments get a
// synthetic parcel ID, office searches return a fixed catalogue, and tracking
// reports a status derived purely from how long ago the parcel was
// "created". This lets the delivery-method checkout flow, shipment creation
// and the tracking poller all be exercised in dev without a carrier account
// or real parcels.
//
// The parcel ID carries its own creation timestamp (see fakeParcelPrefix) so
// tracking stays stateless — no in-memory map to lose across restarts, and
// each poll independently recomputes the status from the parcel's age.
type FakeSpeedyClient struct {
	// now is injectable so tests can drive the tracking progression
	// deterministically; defaults to time.Now.
	now func() time.Time
}

func NewFakeSpeedyClient() *FakeSpeedyClient {
	return &FakeSpeedyClient{now: time.Now}
}

const (
	fakeParcelPrefix   = "DEVP"
	fakeShipmentPrefix = "DEVS"
)

// Tracking progression thresholds, keyed off parcel age. Deliberately short
// so that with a matching FULFILLMENT_POLL_INTERVAL an order visibly moves
// picked_up -> in_transit -> out_for_delivery -> delivered within a couple of
// minutes. The operation codes are the ones domain.FriendlyStatus recognises
// (see .ai/speedy-docs/10-reference.md and domain/tracking.go).
const (
	fakePickedUpAfter       = 0 * time.Second
	fakeInTransitAfter      = 30 * time.Second
	fakeOutForDeliveryAfter = 90 * time.Second
	fakeDeliveredAfter      = 150 * time.Second
)

func (c *FakeSpeedyClient) clock() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

// CreateShipment fabricates a shipment/parcel reference. The parcel ID embeds
// the creation unix time so Track can later derive an age without any stored
// state.
func (c *FakeSpeedyClient) CreateShipment(ctx context.Context, req application.CreateShipmentRequest) (application.ShipmentResult, error) {
	created := c.clock().Unix()
	suffix := rand.Intn(10000) //nolint:gosec // non-cryptographic dev id
	return application.ShipmentResult{
		ShipmentID: fmt.Sprintf("%s-%d-%04d", fakeShipmentPrefix, created, suffix),
		ParcelID:   fmt.Sprintf("%s-%d-%04d", fakeParcelPrefix, created, suffix),
	}, nil
}

// Calculate returns a canned price per requested service so the shipping-cost
// quote flow can be exercised in dev — a flat base plus a per-kg component,
// nudged by the service id so the services don't all price identically.
func (c *FakeSpeedyClient) Calculate(ctx context.Context, req application.CalculateRequest) ([]application.CalculationResult, error) {
	results := make([]application.CalculationResult, 0, len(req.ServiceIDs))
	for _, id := range req.ServiceIDs {
		minor := int64(300) + id%100 + int64(req.WeightKg*100)
		results = append(results, application.CalculationResult{
			ServiceID: id,
			Amount:    money.Money{AmountMinor: minor, Currency: "EUR"},
		})
	}
	return results, nil
}

// Track derives each parcel's latest operation from its embedded creation
// time. Parcel IDs that aren't in the fake format are skipped, so a database
// left over from real-mode use doesn't cause spurious updates.
func (c *FakeSpeedyClient) Track(ctx context.Context, creds application.Credentials, parcelIDs []string) ([]application.TrackedParcel, error) {
	now := c.clock()
	result := make([]application.TrackedParcel, 0, len(parcelIDs))
	for _, id := range parcelIDs {
		created, ok := parseFakeParcelTime(id)
		if !ok {
			continue
		}
		code, desc := fakeOperationForAge(now.Sub(created))
		result = append(result, application.TrackedParcel{
			ParcelID:      id,
			OperationCode: code,
			Description:   desc,
		})
	}
	return result, nil
}

// SearchOffices returns a small fixed catalogue so the checkout office/locker
// picker has something to render. The requested type is honoured; nothing is
// looked up remotely.
func (c *FakeSpeedyClient) SearchOffices(ctx context.Context, creds application.Credentials, siteID int64, name, officeType string) ([]application.Office, error) {
	if officeType == "" {
		officeType = "APT"
	}
	kind := "Office"
	if officeType == "APT" {
		kind = "Locker"
	}
	all := []application.Office{
		{ID: "1", Name: fmt.Sprintf("Central %s (DEV)", kind), Type: officeType},
		{ID: "2", Name: fmt.Sprintf("Mall %s (DEV)", kind), Type: officeType},
		{ID: "3", Name: fmt.Sprintf("Station %s (DEV)", kind), Type: officeType},
	}
	return filterByName(all, name, func(o application.Office) string { return o.Name }), nil
}

// fakeSites is a tiny canned catalogue of Bulgarian cities so profile/checkout
// typeahead resolves without a real Speedy account under SPEEDY_MODE=fake.
var fakeSites = []application.Site{
	{ID: 68134, Name: "Sofia", Type: "gr.", Municipality: "Stolichna", Region: "Sofia (stolitsa)", PostCode: "1000"},
	{ID: 10135, Name: "Plovdiv", Type: "gr.", Municipality: "Plovdiv", Region: "Plovdiv", PostCode: "4000"},
	{ID: 10007, Name: "Varna", Type: "gr.", Municipality: "Varna", Region: "Varna", PostCode: "9000"},
	{ID: 10003, Name: "Burgas", Type: "gr.", Municipality: "Burgas", Region: "Burgas", PostCode: "8000"},
}

func (c *FakeSpeedyClient) SearchSites(ctx context.Context, creds application.Credentials, name string) ([]application.Site, error) {
	return filterByName(fakeSites, name, func(s application.Site) string { return s.Name }), nil
}

func (c *FakeSpeedyClient) SearchComplexes(ctx context.Context, creds application.Credentials, siteID int64, name string) ([]application.Complex, error) {
	all := []application.Complex{
		{ID: 1, Name: "Mladost 1", Type: "zh.k."},
		{ID: 2, Name: "Lyulin 3", Type: "zh.k."},
		{ID: 3, Name: "Lozenets", Type: "kv."},
	}
	return filterByName(all, name, func(x application.Complex) string { return x.Name }), nil
}

func (c *FakeSpeedyClient) SearchStreets(ctx context.Context, creds application.Credentials, siteID int64, name string) ([]application.Street, error) {
	all := []application.Street{
		{ID: 100, Name: "Vitosha", Type: "bul."},
		{ID: 101, Name: "Cherni vrah", Type: "bul."},
		{ID: 102, Name: "Alabin", Type: "ul."},
	}
	return filterByName(all, name, func(s application.Street) string { return s.Name }), nil
}

// filterByName narrows a canned slice by a case-insensitive name-fragment
// match, mirroring how the real Location API ranks by the supplied name.
func filterByName[T any](items []T, name string, nameOf func(T) string) []T {
	q := strings.ToLower(strings.TrimSpace(name))
	if q == "" {
		return items
	}
	out := make([]T, 0, len(items))
	for _, it := range items {
		if strings.Contains(strings.ToLower(nameOf(it)), q) {
			out = append(out, it)
		}
	}
	return out
}

func fakeOperationForAge(age time.Duration) (int, string) {
	switch {
	case age >= fakeDeliveredAfter:
		return 14, "Delivered (dev)"
	case age >= fakeOutForDeliveryAfter:
		return 12, "Out for delivery (dev)"
	case age >= fakeInTransitAfter:
		return 1, "In transit (dev)"
	default:
		return 39, "Picked up (dev)"
	}
}

// parseFakeParcelTime extracts the creation time embedded in a fake parcel ID
// of the form "DEVP-<unix>-<suffix>".
func parseFakeParcelTime(parcelID string) (time.Time, bool) {
	if !strings.HasPrefix(parcelID, fakeParcelPrefix+"-") {
		return time.Time{}, false
	}
	parts := strings.Split(parcelID, "-")
	if len(parts) != 3 {
		return time.Time{}, false
	}
	unix, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(unix, 0), true
}
