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

type InvoiceLineItem struct {
	ID                    uuid.UUID
	InvoiceID             uuid.UUID
	ProductName           string
	VariantLabel          string
	NKSCode               string
	Quantity              int
	UnitPriceInclVAT      money.Money
	UnitPriceExclVAT      money.Money
	VATPerUnit            money.Money
	LineTotalInclVAT      money.Money
	LineTotalExclVAT      money.Money
	LineVATAmount         money.Money
	SortOrder             int
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
