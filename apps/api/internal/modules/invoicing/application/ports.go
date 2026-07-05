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
	GetSettings(ctx context.Context) (domain.InvoiceSettings, error)
	SaveSettings(ctx context.Context, settings domain.InvoiceSettings) error
	ListCouriers(ctx context.Context) ([]domain.Courier, error)
	CreateCourier(ctx context.Context, courier domain.Courier) (*domain.Courier, error)
	UpdateCourier(ctx context.Context, id uuid.UUID, courier domain.Courier) (*domain.Courier, error)
	DeleteCourier(ctx context.Context, id uuid.UUID) error
	LogAuditEvent(ctx context.Context, invoiceNumber, eventType, actor string, metadata map[string]any) error
}

type OrderReader interface {
	FindByID(ctx context.Context, id uuid.UUID) (*ordersdomain.Order, error)
}

type ProductNKSReader interface {
	GetNKSCode(ctx context.Context, productID uuid.UUID) (string, error)
}
