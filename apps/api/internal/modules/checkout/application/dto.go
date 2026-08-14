package application

import (
	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/checkout/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

// PlaceOrderResult is what PlaceOrder returns: exactly one of Order (a fully
// placed pay-on-delivery order) or PaymentRequired (an online-card order
// awaiting widget payment) is set.
type PlaceOrderResult struct {
	Order           *OrderResult
	PaymentRequired *PaymentInitiation
}

// PaymentInitiation carries what the storefront needs to mount the Revolut
// widget for an online-card order that isn't paid yet.
type PaymentInitiation struct {
	OrderID           uuid.UUID
	OrderNumber       string
	RevolutOrderID    string
	RevolutOrderToken string
	Amount            money.Money
	PaymentMethod     string
	Status            string
}

type ContactInput struct {
	FullName string
	Email    string
	Phone    string
}

type AddressInput struct {
	RecipientName string
	Phone         string

	CountryCode string
	CountryID   int64
	SiteID      int64
	City        string
	PostCode    string
	ComplexID   int64
	ComplexName string
	StreetID    int64
	StreetName  string
	StreetNo    string
	BlockNo     string
	EntranceNo  string
	FloorNo     string
	ApartmentNo string
}

func (a AddressInput) toDomain() domain.Address {
	countryCode := a.CountryCode
	if countryCode == "" {
		countryCode = "BG"
	}
	countryID := a.CountryID
	if countryID == 0 {
		countryID = 100
	}
	return domain.Address{
		RecipientName: a.RecipientName, Phone: a.Phone,
		CountryCode: countryCode, CountryID: countryID, SiteID: a.SiteID,
		City: a.City, PostCode: a.PostCode,
		ComplexID: a.ComplexID, ComplexName: a.ComplexName,
		StreetID: a.StreetID, StreetName: a.StreetName, StreetNo: a.StreetNo,
		BlockNo: a.BlockNo, EntranceNo: a.EntranceNo, FloorNo: a.FloorNo, ApartmentNo: a.ApartmentNo,
	}
}

// PlaceOrderInput is the full checkout submission: who's ordering, where
// to ship/bill, how to deliver, and how to pay. Contact is required only
// for guest checkout (no authenticated principal) — the HTTP layer fills
// it from the signed-in profile otherwise.
type PlaceOrderInput struct {
	Contact          ContactInput
	ShippingAddress  AddressInput
	BillingAddress   AddressInput
	DeliveryMethod   string
	DeliveryOfficeID string
	PaymentMethod    string
	DiscountCode     string
	// Locale is the display language the customer is checking out in (e.g.
	// "en", "bg"), sent by the storefront and persisted on the order so a
	// guest's transactional email can honour their language.
	Locale string
}
