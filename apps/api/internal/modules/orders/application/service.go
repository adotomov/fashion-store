package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/orders/domain"
)

type Service struct {
	repo     Repository
	invoices InvoiceGateway
	logger   *slog.Logger
}

func NewService(repo Repository, invoices InvoiceGateway, logger *slog.Logger) *Service {
	return &Service{repo: repo, invoices: invoices, logger: logger}
}

func (s *Service) ListOrders(ctx context.Context, userID uuid.UUID) ([]domain.Order, error) {
	return s.repo.ListByUser(ctx, userID)
}

// CreateOrder is called by the checkout flow (via its OrderGateway port)
// once payment has either succeeded or wasn't required upfront, and by
// tests/seed data through the same path.
func (s *Service) CreateOrder(ctx context.Context, userID uuid.UUID, input CreateOrderInput) (*domain.Order, error) {
	status := domain.OrderStatus(input.Status)
	if !status.Valid() {
		return nil, domain.ValidationError("status is invalid")
	}

	order := domain.Order{
		UserID:           userID,
		OrderNumber:      input.OrderNumber,
		Status:           status,
		Total:            input.Total,
		PlacedAt:         input.PlacedAt,
		ContactName:      input.ContactName,
		ContactEmail:     input.ContactEmail,
		ContactPhone:     input.ContactPhone,
		ShippingAddress:  input.ShippingAddress,
		BillingAddress:   input.BillingAddress,
		DeliveryMethod:   input.DeliveryMethod,
		DeliveryFee:      input.DeliveryFee,
		PaymentMethod:    input.PaymentMethod,
		DeliveryOfficeID: input.DeliveryOfficeID,
		ReservationID:    input.ReservationID,
		DiscountCode:     input.DiscountCode,
		DiscountAmount:   input.DiscountAmount,
	}
	if input.Payment != nil {
		order.Payment = &domain.OrderPayment{
			Provider:          input.Payment.Provider,
			ProviderReference: input.Payment.ProviderReference,
			Status:            input.Payment.Status,
			Amount:            input.Payment.Amount,
		}
	}
	for _, item := range input.Items {
		order.Items = append(order.Items, domain.OrderItem{
			ProductID:    item.ProductID,
			ProductName:  item.ProductName,
			VariantLabel: item.VariantLabel,
			Quantity:     item.Quantity,
			UnitPrice:    item.UnitPrice,
		})
	}
	return s.repo.Create(ctx, order)
}

// CountOrdersByUser is exposed to other modules (via an adapter) so they can
// show order counts without importing this module's domain/repository.
func (s *Service) CountOrdersByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.CountByUser(ctx, userID)
}

func (s *Service) AdminListOrders(ctx context.Context, filter AdminListOrdersFilter) ([]domain.Order, error) {
	return s.repo.AdminList(ctx, filter)
}

// AdminGetOrder returns the order and marks it as viewed by an admin —
// viewing the detail page is what clears it from the "unread" badge.
func (s *Service) FindByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) AdminGetOrder(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	order, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order.ViewedByAdminAt == nil {
		_ = s.repo.MarkViewed(ctx, id)
		order, err = s.repo.FindByID(ctx, id)
		if err != nil {
			return nil, err
		}
	}
	return order, nil
}

func (s *Service) UpdateFulfillment(ctx context.Context, id uuid.UUID, input UpdateFulfillmentInput) (*domain.Order, error) {
	if input.Status != nil && !domain.OrderStatus(*input.Status).Valid() {
		return nil, domain.ValidationError("status is invalid")
	}
	order, err := s.repo.UpdateFulfillment(ctx, id, input)
	if err != nil {
		return nil, err
	}
	// Generate invoice for COD/EasyBox orders upon delivery confirmation.
	if input.Status != nil && *input.Status == string(domain.OrderStatusDelivered) && s.invoices != nil {
		if err := s.invoices.GenerateForOrder(ctx, id); err != nil {
			s.logger.Error("failed to generate invoice for delivered order", "error", err, "order_id", id)
		}
	}
	return order, nil
}

func (s *Service) CountUnviewedOrders(ctx context.Context) (int, error) {
	return s.repo.CountUnviewed(ctx)
}

func (s *Service) OrderStats(ctx context.Context, since time.Time) (OrderStats, error) {
	return s.repo.Stats(ctx, since)
}

func (s *Service) ListAwaitingTracking(ctx context.Context) ([]domain.Order, error) {
	return s.repo.ListAwaitingTracking(ctx)
}
