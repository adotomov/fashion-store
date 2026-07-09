package domain

import (
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

type DocumentType string

const (
	DocumentTypeFaktura DocumentType = "фактура"
	DocumentTypeStorno  DocumentType = "сторно"
)

type InvoiceSettings struct {
	CompanyName             string
	CompanyLegalType        string
	CompanyEIK              string
	CompanyAddressStreet     string
	CompanyAddressCity       string
	CompanyAddressPostalCode string
	CompanyAddressCountry    string
	CompanyEmail            string
	CompanyPhone            string
	NRAStoreNumber          string
	VATNumber               string
	VATRate                 float64
}

type Courier struct {
	ID         uuid.UUID
	Name       string
	Identifier string
	IsActive   bool
	SortOrder  int
	CreatedAt  time.Time
}

// TaxGroup is a Bulgarian VAT fiscal group (identifiers А–Ж). Products
// reference one; its rate drives per-line VAT on generated invoices.
type TaxGroup struct {
	ID         uuid.UUID
	Identifier string
	VATRate    float64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type InvoiceLineItem struct {
	ID                    uuid.UUID
	InvoiceID             uuid.UUID
	ProductName           string
	VariantLabel          string
	Quantity              int
	UnitPriceInclVAT      money.Money
	UnitPriceExclVAT      money.Money
	VATPerUnit            money.Money
	LineTotalInclVAT      money.Money
	LineTotalExclVAT      money.Money
	LineVATAmount         money.Money
	// VATRate is the percentage rate used for this line's VAT split, taken
	// from the product's tax group at issuance time (snapshotted).
	VATRate   float64
	SortOrder int
}

type Invoice struct {
	ID                  uuid.UUID
	InvoiceNumber       string
	DocumentType        DocumentType
	OrderID             uuid.UUID
	StornoOfInvoiceID   *uuid.UUID

	OrderNumber string
	PlacedAt    time.Time // stored in UTC, displayed in Europe/Sofia

	PaymentMethod         string
	CardProvider          *string
	CardProviderReference *string
	CourierName           *string
	CourierIdentifier     *string

	// Seller snapshot (frozen at issuance)
	CompanyName      string
	CompanyLegalType string
	CompanyEIK       string
	CompanyAddress   string
	CompanyEmail     string
	CompanyPhone     string
	NRAStoreNumber   string
	VATNumber        string
	VATRate          float64

	// Buyer snapshot
	RecipientName    string
	RecipientAddress string
	RecipientEmail   string

	// Monetary totals
	SubtotalExclVAT money.Money
	VATAmount       money.Money
	TotalInclVAT    money.Money
	DeliveryFee     money.Money
	DiscountAmount  *money.Money

	LineItems []InvoiceLineItem
	CreatedAt time.Time
}
