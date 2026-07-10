package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/invoicing/domain"
	ordersdomain "github.com/adotomov/fashion-store/apps/api/internal/modules/orders/domain"
)

type Repository interface {
	CreateInvoice(ctx context.Context, invoice domain.Invoice) (*domain.Invoice, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error)
	FindByOrderID(ctx context.Context, orderID uuid.UUID) (*domain.Invoice, error)
	List(ctx context.Context, filter ListFilter) ([]domain.Invoice, error)
	CountInvoices(ctx context.Context, filter ListFilter) (int, error)
	GetSettings(ctx context.Context) (domain.InvoiceSettings, error)
	SaveSettings(ctx context.Context, settings domain.InvoiceSettings) error
	ListCouriers(ctx context.Context) ([]domain.Courier, error)
	CreateCourier(ctx context.Context, courier domain.Courier) (*domain.Courier, error)
	UpdateCourier(ctx context.Context, id uuid.UUID, courier domain.Courier) (*domain.Courier, error)
	DeleteCourier(ctx context.Context, id uuid.UUID) error
	ListTaxGroups(ctx context.Context) ([]domain.TaxGroup, error)
	CreateTaxGroup(ctx context.Context, group domain.TaxGroup) (*domain.TaxGroup, error)
	UpdateTaxGroup(ctx context.Context, id uuid.UUID, group domain.TaxGroup) (*domain.TaxGroup, error)
	DeleteTaxGroup(ctx context.Context, id uuid.UUID) error
	LogAuditEvent(ctx context.Context, invoiceNumber, eventType, actor string, metadata map[string]any) error
}

type OrderReader interface {
	FindByID(ctx context.Context, id uuid.UUID) (*ordersdomain.Order, error)
}

// ProductInvoiceReader exposes the product facts the invoice generator needs
// from the catalog module: the assigned tax group (nil when the product has
// none, in which case the default rate applies).
type ProductInvoiceReader interface {
	GetTaxGroupID(ctx context.Context, productID uuid.UUID) (*uuid.UUID, error)
}
