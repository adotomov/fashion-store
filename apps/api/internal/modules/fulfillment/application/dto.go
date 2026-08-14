package application

import "github.com/adotomov/fashion-store/apps/api/internal/shared/money"

// Credentials is the subset of a provider's config needed to authenticate a
// single API call — resolved from the stored config map at call time so a
// credential change takes effect on the next call without restarting
// anything.
type Credentials struct {
	Username       string
	Password       string
	Language       string
	ClientSystemID string
}

// ShipmentRecipient covers both delivery shapes Speedy supports for our
// checkout: a door-delivery address, or a locker/office ID picked by the
// customer. Exactly one of (address fields) or OfficeID is set.
//
// The address is expressed with Speedy's resolved location codes (site,
// complex, street) rather than free text, so the carrier never has to
// guess-resolve a typed address. CountryID is Speedy's numeric country code
// (Bulgaria = 100).
type ShipmentRecipient struct {
	ContactName string
	Phone       string
	Email       string

	CountryID   int64
	SiteID      int64
	ComplexID   int64
	StreetID    int64
	StreetNo    string
	BlockNo     string
	EntranceNo  string
	FloorNo     string
	ApartmentNo string

	OfficeID string
}

type CreateShipmentRequest struct {
	Creds          Credentials
	ServiceID      string
	ParcelWeightKg float64
	Recipient      ShipmentRecipient
	CODAmount      money.Money
	RequireCOD     bool
	Ref1           string
}

type ShipmentResult struct {
	ShipmentID string
	ParcelID   string
}

// CalculateRequest asks Speedy to price a single parcel of the given weight to
// a destination site, for one or more services at once. Sender is intentionally
// unset — Speedy uses the authenticated account's default pickup location.
type CalculateRequest struct {
	Creds      Credentials
	SiteID     int64
	WeightKg   float64
	ServiceIDs []int64
}

// CalculationResult is one priced service from a CalculateRequest; services
// that erred (not available for the destination) are omitted.
type CalculationResult struct {
	ServiceID int64
	Amount    money.Money
}

// ShippingCostQuote is a priced delivery method for an order — the internal,
// never-displayed record of what Speedy would charge.
type ShippingCostQuote struct {
	DeliveryMethod string
	ServiceID      string
	Amount         money.Money
}

// TrackedParcel is one parcel's latest tracking operation — only the latest
// is needed since the poller just keeps the order's status fresh, not a full
// history.
type TrackedParcel struct {
	ParcelID      string
	OperationCode int
	Description   string
}

type Office struct {
	ID   string
	Name string
	Type string
}

// Site is a Speedy populated place (city/town/village) resolved from the
// Location API. ID is the siteId used when building a structured address.
type Site struct {
	ID           int64
	Name         string
	Type         string
	Municipality string
	Region       string
	PostCode     string
}

// Complex is a Speedy residential complex / quarter (кв./жк.) within a site.
type Complex struct {
	ID   int64
	Name string
	Type string
}

// Street is a Speedy street within a site.
type Street struct {
	ID   int64
	Name string
	Type string
}

// CreateShipmentInput is what checkout (via the FulfillmentGateway port)
// hands over after an order is created.
type CreateShipmentInput struct {
	Provider       string
	DeliveryMethod string

	ContactName string
	Phone       string
	Email       string

	CountryID   int64
	SiteID      int64
	ComplexID   int64
	StreetID    int64
	StreetNo    string
	BlockNo     string
	EntranceNo  string
	FloorNo     string
	ApartmentNo string
	OfficeID    string

	RequireCOD bool
	CODAmount  money.Money
	Ref1       string

	// WeightKg is the real parcel weight summed from the order's SKUs. When 0
	// (no SKU carried a weight) the provider's configured default is used.
	WeightKg float64
}
