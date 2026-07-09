package application

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/invoicing/domain"
	ordersdomain "github.com/adotomov/fashion-store/apps/api/internal/modules/orders/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

type Service struct {
	repo    Repository
	orders  OrderReader
	catalog ProductInvoiceReader
}

func NewService(repo Repository, orders OrderReader, catalog ProductInvoiceReader) *Service {
	return &Service{repo: repo, orders: orders, catalog: catalog}
}

// GenerateForOrder creates a фактура invoice for the given order. It is
// idempotent: if an invoice already exists for the order, the existing one is
// returned without creating a duplicate.
func (s *Service) GenerateForOrder(ctx context.Context, orderID uuid.UUID, actor string) (*domain.Invoice, error) {
	existing, err := s.repo.FindByOrderID(ctx, orderID)
	if err != nil && !errors.Is(err, domain.ErrInvoiceNotFound) {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("load order: %w", err)
	}

	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("load invoice settings: %w", err)
	}
	if settings.CompanyEIK == "" || settings.NRAStoreNumber == "" {
		return nil, domain.ErrSettingsIncomplete
	}

	lineItems, grandTotalIncl, grandTotalExcl, grandVAT := s.buildLineItems(ctx, order)

	recipientAddr := buildAddressString(order.ShippingAddress)

	inv := domain.Invoice{
		ID:           uuid.New(),
		DocumentType: domain.DocumentTypeFaktura,
		OrderID:      orderID,
		OrderNumber:  order.OrderNumber,
		PlacedAt:     order.PlacedAt,
		PaymentMethod: order.PaymentMethod,

		CompanyName:      settings.CompanyName,
		CompanyLegalType: settings.CompanyLegalType,
		CompanyEIK:       settings.CompanyEIK,
		CompanyAddress:   formatSettingsAddress(settings),
		CompanyEmail:     settings.CompanyEmail,
		CompanyPhone:     settings.CompanyPhone,
		NRAStoreNumber:   settings.NRAStoreNumber,
		VATNumber:        settings.VATNumber,
		VATRate:          settings.VATRate,

		RecipientName:    order.ContactName,
		RecipientAddress: recipientAddr,
		RecipientEmail:   order.ContactEmail,

		SubtotalExclVAT: money.Money{AmountMinor: grandTotalExcl, Currency: order.Total.Currency},
		VATAmount:       money.Money{AmountMinor: grandVAT, Currency: order.Total.Currency},
		TotalInclVAT:    money.Money{AmountMinor: grandTotalIncl, Currency: order.Total.Currency},
		DeliveryFee:     order.DeliveryFee,
		DiscountAmount:  order.DiscountAmount,

		LineItems: lineItems,
	}

	// Payment reference
	switch order.PaymentMethod {
	case ordersdomain.PaymentMethodCardOnline:
		if order.Payment != nil {
			inv.CardProvider = &order.Payment.Provider
			inv.CardProviderReference = &order.Payment.ProviderReference
		}
	case ordersdomain.PaymentMethodCashOnDelivery, ordersdomain.PaymentMethodCardOnEasyBox:
		couriers, _ := s.repo.ListCouriers(ctx)
		if carrier := order.Carrier; carrier != nil {
			inv.CourierName = carrier
			for _, c := range couriers {
				if strings.EqualFold(c.Name, *carrier) || strings.EqualFold(c.Identifier, *carrier) {
					inv.CourierIdentifier = &c.Identifier
					break
				}
			}
		}
	}

	created, err := s.repo.CreateInvoice(ctx, inv)
	if err != nil {
		return nil, fmt.Errorf("create invoice: %w", err)
	}

	_ = s.repo.LogAuditEvent(ctx, created.InvoiceNumber, "invoice.created", actor, nil)
	return created, nil
}

// GenerateStorno issues a сторно invoice referencing the original фактура.
func (s *Service) GenerateStorno(ctx context.Context, invoiceID uuid.UUID, actor string) (*domain.Invoice, error) {
	original, err := s.repo.FindByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if original.DocumentType == domain.DocumentTypeStorno {
		return nil, domain.ErrCannotStornoAStorno
	}

	storno := *original
	storno.ID = uuid.New()
	storno.InvoiceNumber = "" // assigned by DB sequence default
	storno.DocumentType = domain.DocumentTypeStorno
	storno.StornoOfInvoiceID = &original.ID
	storno.CreatedAt = time.Time{}

	created, err := s.repo.CreateInvoice(ctx, storno)
	if err != nil {
		return nil, fmt.Errorf("create storno invoice: %w", err)
	}

	_ = s.repo.LogAuditEvent(ctx, created.InvoiceNumber, "invoice.storno_issued", actor, map[string]any{
		"original_invoice_number": original.InvoiceNumber,
	})
	return created, nil
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]domain.Invoice, error) {
	return s.repo.List(ctx, filter)
}

func (s *Service) FindByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) GetSettings(ctx context.Context) (domain.InvoiceSettings, error) {
	return s.repo.GetSettings(ctx)
}

func (s *Service) SaveSettings(ctx context.Context, settings domain.InvoiceSettings) error {
	return s.repo.SaveSettings(ctx, settings)
}

func (s *Service) ListCouriers(ctx context.Context) ([]domain.Courier, error) {
	return s.repo.ListCouriers(ctx)
}

func (s *Service) CreateCourier(ctx context.Context, courier domain.Courier) (*domain.Courier, error) {
	courier.ID = uuid.New()
	return s.repo.CreateCourier(ctx, courier)
}

func (s *Service) UpdateCourier(ctx context.Context, id uuid.UUID, courier domain.Courier) (*domain.Courier, error) {
	return s.repo.UpdateCourier(ctx, id, courier)
}

func (s *Service) DeleteCourier(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteCourier(ctx, id)
}

func (s *Service) ListTaxGroups(ctx context.Context) ([]domain.TaxGroup, error) {
	return s.repo.ListTaxGroups(ctx)
}

func (s *Service) CreateTaxGroup(ctx context.Context, group domain.TaxGroup) (*domain.TaxGroup, error) {
	if !validTaxGroup(group) {
		return nil, domain.ErrInvalidTaxGroup
	}
	group.ID = uuid.New()
	return s.repo.CreateTaxGroup(ctx, group)
}

func (s *Service) UpdateTaxGroup(ctx context.Context, id uuid.UUID, group domain.TaxGroup) (*domain.TaxGroup, error) {
	if !validTaxGroup(group) {
		return nil, domain.ErrInvalidTaxGroup
	}
	return s.repo.UpdateTaxGroup(ctx, id, group)
}

func (s *Service) DeleteTaxGroup(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteTaxGroup(ctx, id)
}

func validTaxGroup(g domain.TaxGroup) bool {
	return domain.ValidTaxGroupIdentifier(g.Identifier) && g.VATRate >= 0 && g.VATRate <= 100
}

// ExportCSV returns a CSV byte slice for invoices in the given date range.
func (s *Service) ExportCSV(ctx context.Context, from, to time.Time, actor string) ([]byte, error) {
	invoices, err := s.repo.List(ctx, ListFilter{From: &from, To: &to, Limit: 10000})
	if err != nil {
		return nil, err
	}

	loc, _ := time.LoadLocation("Europe/Sofia")

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{
		"invoice_number", "document_type", "order_number", "date_sofia",
		"payment_method", "recipient_name", "subtotal_excl_vat", "vat_amount",
		"total_incl_vat", "currency", "delivery_fee",
	})
	for _, inv := range invoices {
		dateSofia := inv.CreatedAt.In(loc).Format("2006-01-02 15:04:05")
		_ = w.Write([]string{
			inv.InvoiceNumber,
			string(inv.DocumentType),
			inv.OrderNumber,
			dateSofia,
			inv.PaymentMethod,
			inv.RecipientName,
			formatMinor(inv.SubtotalExclVAT.AmountMinor),
			formatMinor(inv.VATAmount.AmountMinor),
			formatMinor(inv.TotalInclVAT.AmountMinor),
			inv.TotalInclVAT.Currency,
			formatMinor(inv.DeliveryFee.AmountMinor),
		})
	}
	w.Flush()

	_ = s.repo.LogAuditEvent(ctx, "*", "invoice.csv_exported", actor, map[string]any{
		"from": from.Format(time.RFC3339),
		"to":   to.Format(time.RFC3339),
		"rows": len(invoices),
	})
	return buf.Bytes(), nil
}

func (s *Service) LogPDFView(ctx context.Context, inv *domain.Invoice, actor string) {
	_ = s.repo.LogAuditEvent(ctx, inv.InvoiceNumber, "invoice.pdf_viewed", actor, nil)
}

// defaultVATRate is used for lines whose product has no tax group assigned
// (and for the delivery fee) — it matches the historical hardcoded 20% split.
const defaultVATRate = 20.0

// buildLineItems computes per-line VAT breakdown for all order items. Each
// line's rate comes from its product's tax group (falling back to 20% when
// none is assigned): excl = floor(incl * 100 / (100 + rate)), vat = incl - excl.
func (s *Service) buildLineItems(ctx context.Context, order *ordersdomain.Order) (
	items []domain.InvoiceLineItem, grandInclMinor, grandExclMinor, grandVATMinor int64,
) {
	rates := s.taxRatesByGroupID(ctx)

	for i, oi := range order.Items {
		rate := defaultVATRate
		if oi.ProductID != nil {
			if groupID, err := s.catalog.GetTaxGroupID(ctx, *oi.ProductID); err == nil && groupID != nil {
				if r, ok := rates[*groupID]; ok {
					rate = r
				}
			}
		}

		unitIncl := oi.UnitPrice.AmountMinor
		unitExcl := vatExclusive(unitIncl, rate)
		vatUnit := unitIncl - unitExcl

		qty := int64(oi.Quantity)
		lineIncl := unitIncl * qty
		lineExcl := unitExcl * qty
		lineVAT := vatUnit * qty

		grandInclMinor += lineIncl
		grandExclMinor += lineExcl
		grandVATMinor += lineVAT

		items = append(items, domain.InvoiceLineItem{
			ID:                uuid.New(),
			ProductName:       oi.ProductName,
			VariantLabel:      oi.VariantLabel,
			Quantity:          oi.Quantity,
			UnitPriceInclVAT:  money.Money{AmountMinor: unitIncl, Currency: oi.UnitPrice.Currency},
			UnitPriceExclVAT:  money.Money{AmountMinor: unitExcl, Currency: oi.UnitPrice.Currency},
			VATPerUnit:        money.Money{AmountMinor: vatUnit, Currency: oi.UnitPrice.Currency},
			LineTotalInclVAT:  money.Money{AmountMinor: lineIncl, Currency: oi.UnitPrice.Currency},
			LineTotalExclVAT:  money.Money{AmountMinor: lineExcl, Currency: oi.UnitPrice.Currency},
			LineVATAmount:     money.Money{AmountMinor: lineVAT, Currency: oi.UnitPrice.Currency},
			VATRate:           rate,
			SortOrder:         i,
		})
	}

	// Add delivery fee to grand totals (also VAT-inclusive, at the default rate)
	if order.DeliveryFee.AmountMinor > 0 {
		df := order.DeliveryFee.AmountMinor
		dfExcl := vatExclusive(df, defaultVATRate)
		grandInclMinor += df
		grandExclMinor += dfExcl
		grandVATMinor += df - dfExcl
	}

	// Subtract discount from incl total; split proportionally for excl/VAT
	if order.DiscountAmount != nil && order.DiscountAmount.AmountMinor > 0 {
		disc := order.DiscountAmount.AmountMinor
		discExcl := vatExclusive(disc, defaultVATRate)
		grandInclMinor -= disc
		grandExclMinor -= discExcl
		grandVATMinor -= disc - discExcl
	}

	return items, grandInclMinor, grandExclMinor, grandVATMinor
}

// vatExclusive strips VAT from a VAT-inclusive minor amount at the given rate:
// excl = floor(incl * 100 / (100 + rate)). A rate of 20 yields incl*100/120,
// identical to the previous hardcoded behavior.
func vatExclusive(inclMinor int64, rate float64) int64 {
	denom := int64(100 + rate)
	if denom <= 0 {
		return inclMinor
	}
	return inclMinor * 100 / denom
}

// taxRatesByGroupID loads all tax groups once into a rate lookup for the
// duration of a single invoice build. Errors are swallowed to the default
// rate — a missing group simply means lines fall back to 20%.
func (s *Service) taxRatesByGroupID(ctx context.Context) map[uuid.UUID]float64 {
	rates := map[uuid.UUID]float64{}
	groups, err := s.repo.ListTaxGroups(ctx)
	if err != nil {
		return rates
	}
	for _, g := range groups {
		rates[g.ID] = g.VATRate
	}
	return rates
}

func formatSettingsAddress(s domain.InvoiceSettings) string {
	var parts []string
	if s.CompanyAddressStreet != "" {
		parts = append(parts, s.CompanyAddressStreet)
	}
	cityPart := strings.TrimSpace(s.CompanyAddressPostalCode + " " + s.CompanyAddressCity)
	if cityPart != "" {
		parts = append(parts, cityPart)
	}
	if s.CompanyAddressCountry != "" {
		parts = append(parts, s.CompanyAddressCountry)
	}
	return strings.Join(parts, ", ")
}

func buildAddressString(addr ordersdomain.OrderAddress) string {
	parts := []string{addr.Line1}
	if addr.Line2 != "" {
		parts = append(parts, addr.Line2)
	}
	cityLine := addr.City
	if addr.Region != "" {
		cityLine += ", " + addr.Region
	}
	if addr.PostalCode != "" {
		cityLine += " " + addr.PostalCode
	}
	parts = append(parts, cityLine, addr.CountryCode)
	return strings.Join(parts, ", ")
}

func formatMinor(minor int64) string {
	major := minor / 100
	cents := minor % 100
	if cents < 0 {
		cents = -cents
	}
	return fmt.Sprintf("%d.%02d", major, cents)
}
