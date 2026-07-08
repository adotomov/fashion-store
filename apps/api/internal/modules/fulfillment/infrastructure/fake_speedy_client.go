package infrastructure

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/fulfillment/application"
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
// picker has something to render. The city is echoed into the names and the
// requested type is honoured; nothing is looked up remotely.
func (c *FakeSpeedyClient) SearchOffices(ctx context.Context, creds application.Credentials, city, officeType string) ([]application.Office, error) {
	if officeType == "" {
		officeType = "APT"
	}
	if city == "" {
		city = "Sofia"
	}
	return []application.Office{
		{ID: "1", Name: fmt.Sprintf("%s Central (DEV)", city), Type: officeType},
		{ID: "2", Name: fmt.Sprintf("%s Mall (DEV)", city), Type: officeType},
		{ID: "3", Name: fmt.Sprintf("%s Station (DEV)", city), Type: officeType},
	}, nil
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
