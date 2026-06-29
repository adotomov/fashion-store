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
type ShipmentRecipient struct {
	ContactName string
	Phone       string
	Email       string

	City        string
	PostalCode  string
	Line1       string
	Line2       string
	CountryCode string

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

// CreateShipmentInput is what checkout (via the FulfillmentGateway port)
// hands over after an order is created.
type CreateShipmentInput struct {
	Provider       string
	DeliveryMethod string

	ContactName string
	Phone       string
	Email       string

	City        string
	PostalCode  string
	Line1       string
	Line2       string
	CountryCode string
	OfficeID    string

	RequireCOD bool
	CODAmount  money.Money
	Ref1       string
}
