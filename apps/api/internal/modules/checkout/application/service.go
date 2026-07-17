package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/checkout/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

type Service struct {
	cart        CartGateway
	inventory   InventoryGateway
	users       UserGateway
	orders      OrderGateway
	payments    PaymentGateway
	fulfillment FulfillmentGateway
	discounts   DiscountGateway
	invoices    InvoiceGateway
	events      WebhookEventStore
	logger      *slog.Logger
}

func NewService(cart CartGateway, inventory InventoryGateway, users UserGateway, orders OrderGateway, payments PaymentGateway, fulfillment FulfillmentGateway, discounts DiscountGateway, invoices InvoiceGateway, events WebhookEventStore, logger *slog.Logger) *Service {
	return &Service{cart: cart, inventory: inventory, users: users, orders: orders, payments: payments, fulfillment: fulfillment, discounts: discounts, invoices: invoices, events: events, logger: logger}
}

// ListDeliveryMethods filters out any method whose logistics provider isn't
// enabled yet, so checkout never offers a delivery option nobody can
// actually fulfill.
func (s *Service) ListDeliveryMethods(ctx context.Context) []domain.DeliveryMethod {
	all := domain.DeliveryMethods()
	available := make([]domain.DeliveryMethod, 0, len(all))
	for _, m := range all {
		if s.fulfillment.IsProviderEnabled(ctx, domain.ProviderFor(m.Code)) {
			available = append(available, m)
		}
	}
	return available
}

func generateOrderNumber() string {
	return fmt.Sprintf("ORD-%s-%s", time.Now().UTC().Format("20060102150405"), strings.ToUpper(uuid.New().String()[:6]))
}

func toOrderAddress(a domain.Address) OrderAddress {
	return OrderAddress{
		RecipientName: a.RecipientName, Phone: a.Phone, Line1: a.Line1, Line2: a.Line2,
		City: a.City, Region: a.Region, PostalCode: a.PostalCode, CountryCode: a.CountryCode,
	}
}

// addressFromOrder rebuilds a domain.Address from the persisted order snapshot,
// used when booking the shipment during webhook-driven settlement.
func addressFromOrder(a OrderAddress) domain.Address {
	return domain.Address{
		RecipientName: a.RecipientName, Phone: a.Phone, Line1: a.Line1, Line2: a.Line2,
		City: a.City, Region: a.Region, PostalCode: a.PostalCode, CountryCode: a.CountryCode,
	}
}

// PlaceOrder orchestrates checkout up to the point payment diverges: validate
// the cart and chosen methods, price the order, and reserve stock for every
// line (so nothing oversells while a payment is in flight). Then it branches:
//   - pay-on-delivery methods settle in person, so the reservation is committed
//     and the order placed synchronously here.
//   - online card payment opens a Revolut order and returns its widget token;
//     the order is left pending_payment with stock still reserved, and is
//     settled later by FinalizePaidOrder (webhook) or released by the
//     abandoned-payment sweeper.
func (s *Service) PlaceOrder(ctx context.Context, owner CartOwner, principalUserID *uuid.UUID, input PlaceOrderInput) (PlaceOrderResult, error) {
	cartSnap, err := s.cart.GetCart(ctx, owner)
	if err != nil {
		return PlaceOrderResult{}, err
	}
	if len(cartSnap.Lines) == 0 {
		return PlaceOrderResult{}, domain.ErrCartEmpty
	}

	deliveryMethod, ok := domain.FindDeliveryMethod(input.DeliveryMethod)
	if !ok {
		return PlaceOrderResult{}, domain.ErrInvalidDeliveryMethod
	}
	if !domain.ValidPaymentMethod(input.PaymentMethod) {
		return PlaceOrderResult{}, domain.ErrInvalidPaymentMethod
	}
	if !domain.PaymentMethodAllowedFor(deliveryMethod.Code, input.PaymentMethod) {
		return PlaceOrderResult{}, domain.ErrPaymentMethodNotAllowed
	}
	if !s.fulfillment.IsProviderEnabled(ctx, domain.ProviderFor(deliveryMethod.Code)) {
		return PlaceOrderResult{}, domain.ErrDeliveryMethodUnavailable
	}
	if deliveryMethod.Code == domain.DeliveryMethodEasyBox && input.DeliveryOfficeID == "" {
		return PlaceOrderResult{}, domain.ErrOfficeRequired
	}

	shippingAddr := input.ShippingAddress.toDomain()
	if err := shippingAddr.Validate(); err != nil {
		return PlaceOrderResult{}, err
	}
	billingAddr := input.BillingAddress.toDomain()
	if err := billingAddr.Validate(); err != nil {
		return PlaceOrderResult{}, err
	}

	var userID uuid.UUID
	contactName, contactEmail, contactPhone := input.Contact.FullName, input.Contact.Email, input.Contact.Phone
	if principalUserID != nil {
		userID = *principalUserID
	} else {
		contact := domain.Contact{FullName: contactName, Email: contactEmail, Phone: contactPhone}
		if err := contact.Validate(); err != nil {
			return PlaceOrderResult{}, err
		}
		id, err := s.users.EnsureUser(ctx, contact.Email, contact.FullName)
		if err != nil {
			return PlaceOrderResult{}, err
		}
		userID = id
	}

	var subtotal int64
	currency := deliveryMethod.Fee.Currency
	reserveLines := make([]ReserveLine, 0, len(cartSnap.Lines))
	orderItems := make([]CreateOrderItemInput, 0, len(cartSnap.Lines))
	for _, line := range cartSnap.Lines {
		if line.Quantity > line.AvailableQuantity {
			return PlaceOrderResult{}, domain.ErrInsufficientStock
		}
		subtotal += line.UnitPrice.AmountMinor * int64(line.Quantity)
		currency = line.UnitPrice.Currency
		reserveLines = append(reserveLines, ReserveLine{VariantID: line.VariantID, Quantity: line.Quantity})
		orderItems = append(orderItems, CreateOrderItemInput{
			ProductID:    line.ProductID,
			ProductName:  line.ProductName,
			VariantLabel: line.VariantLabel,
			Quantity:     line.Quantity,
			UnitPrice:    line.UnitPrice,
		})
	}

	// Apply discount code if provided.
	var discountCodeStr *string
	var discountAmount *money.Money
	var discountCodeID uuid.UUID
	if input.DiscountCode != "" {
		info, err := s.discounts.ValidateCode(ctx, input.DiscountCode)
		if err != nil {
			return PlaceOrderResult{}, domain.ErrInvalidDiscountCode
		}
		discountMinor := subtotal * int64(info.ValuePercent) / 100
		da := money.Money{AmountMinor: discountMinor, Currency: currency}
		discountAmount = &da
		code := strings.ToUpper(strings.TrimSpace(input.DiscountCode))
		discountCodeStr = &code
		discountCodeID = info.CodeID
	}

	subtotalAfterDiscount := subtotal
	if discountAmount != nil {
		subtotalAfterDiscount -= discountAmount.AmountMinor
		if subtotalAfterDiscount < 0 {
			subtotalAfterDiscount = 0
		}
	}
	total := money.Money{AmountMinor: subtotalAfterDiscount + deliveryMethod.Fee.AmountMinor, Currency: currency}
	orderNumber := generateOrderNumber()

	reservationID, err := s.inventory.Reserve(ctx, reserveLines, &userID)
	if err != nil {
		return PlaceOrderResult{}, err
	}

	// Online card payment: open a Revolut order, persist the order as
	// pending_payment, and hand the widget token back to the client. Stock stays
	// reserved (not committed) until the payment webhook confirms it via
	// FinalizePaidOrder — the reservation is the safety net if the customer
	// never completes the payment (released by the abandoned-payment sweeper).
	if domain.RequiresUpfrontPayment(input.PaymentMethod) {
		return s.initiateCardPayment(ctx, owner, cardPaymentParams{
			orderNumber:      orderNumber,
			userID:           userID,
			reservationID:    reservationID,
			total:            total,
			contactName:      contactName,
			contactEmail:     contactEmail,
			contactPhone:     contactPhone,
			shippingAddr:     shippingAddr,
			billingAddr:      billingAddr,
			deliveryMethod:   deliveryMethod,
			deliveryOfficeID: input.DeliveryOfficeID,
			paymentMethod:    input.PaymentMethod,
			discountCode:     discountCodeStr,
			discountAmount:   discountAmount,
			discountCodeID:   discountCodeID,
			items:            orderItems,
		})
	}

	// Pay-on-delivery methods settle in person, so the order is placed straight
	// away: commit stock and create the order in its normal pending state.
	if err := s.inventory.Commit(ctx, reservationID, &userID); err != nil {
		_ = s.inventory.Release(ctx, reservationID, &userID)
		return PlaceOrderResult{}, err
	}

	result, err := s.orders.CreateOrder(ctx, CreateOrderInput{
		UserID:           userID,
		OrderNumber:      orderNumber,
		ContactName:      contactName,
		ContactEmail:     contactEmail,
		ContactPhone:     contactPhone,
		ShippingAddress:  toOrderAddress(shippingAddr),
		BillingAddress:   toOrderAddress(billingAddr),
		DeliveryMethod:   deliveryMethod.Code,
		DeliveryFee:      deliveryMethod.Fee,
		DeliveryOfficeID: input.DeliveryOfficeID,
		PaymentMethod:    input.PaymentMethod,
		ReservationID:    reservationID,
		Status:           string(orderStatusPending),
		Total:            total,
		DiscountCode:     discountCodeStr,
		DiscountAmount:   discountAmount,
		Items:            orderItems,
	})
	if err != nil {
		return PlaceOrderResult{}, err
	}

	// Increment discount code use count after successful order creation.
	if discountCodeID != uuid.Nil {
		if err := s.discounts.UseCode(ctx, discountCodeID); err != nil {
			s.logger.Error("failed to increment discount code use", "error", err, "code_id", discountCodeID)
		}
	}

	_ = s.cart.ClearCart(ctx, owner)

	s.createShipment(ctx, result, deliveryMethod, shippingAddr, contactName, contactPhone, contactEmail, input.DeliveryOfficeID, input.PaymentMethod, total)

	// Generate the invoice for every successfully placed order, regardless of
	// payment or delivery method. Runs after shipment booking so COD/EasyBox
	// invoices capture the assigned courier. GenerateForOrder is idempotent, so
	// the orders module's delivered-status trigger stays a harmless retry.
	if s.invoices != nil {
		if err := s.invoices.GenerateForOrder(ctx, result.ID); err != nil {
			s.logger.Error("failed to generate invoice for order", "error", err, "order_id", result.ID)
		}
	}

	return PlaceOrderResult{Order: &result}, nil
}

// order status/payment string constants used across the checkout settlement
// flow. Kept local so checkout doesn't import the orders module's domain.
const (
	orderStatusPending        = "pending"
	orderStatusPendingPayment = "pending_payment"
	orderStatusPaid           = "paid"

	paymentProviderRevolut = "revolut"
	paymentStatusPending   = "pending"
)

type cardPaymentParams struct {
	orderNumber      string
	userID           uuid.UUID
	reservationID    uuid.UUID
	total            money.Money
	contactName      string
	contactEmail     string
	contactPhone     string
	shippingAddr     domain.Address
	billingAddr      domain.Address
	deliveryMethod   domain.DeliveryMethod
	deliveryOfficeID string
	paymentMethod    string
	discountCode     *string
	discountAmount   *money.Money
	discountCodeID   uuid.UUID
	items            []CreateOrderItemInput
}

// initiateCardPayment opens the Revolut order and persists a pending_payment
// order, returning the widget token. Stock is already reserved by the caller;
// on any failure here the reservation is released so it isn't orphaned.
func (s *Service) initiateCardPayment(ctx context.Context, owner CartOwner, p cardPaymentParams) (PlaceOrderResult, error) {
	paymentOrder, err := s.payments.CreateOrder(ctx, CreatePaymentOrderInput{
		Amount:        p.total,
		OrderNumber:   p.orderNumber,
		CustomerEmail: p.contactEmail,
	})
	if err != nil {
		_ = s.inventory.Release(ctx, p.reservationID, &p.userID)
		s.logger.Error("failed to create revolut order", "error", err, "order_number", p.orderNumber)
		return PlaceOrderResult{}, domain.ErrPaymentInitiation
	}

	reservationID := p.reservationID
	result, err := s.orders.CreateOrder(ctx, CreateOrderInput{
		UserID:           p.userID,
		OrderNumber:      p.orderNumber,
		ContactName:      p.contactName,
		ContactEmail:     p.contactEmail,
		ContactPhone:     p.contactPhone,
		ShippingAddress:  toOrderAddress(p.shippingAddr),
		BillingAddress:   toOrderAddress(p.billingAddr),
		DeliveryMethod:   p.deliveryMethod.Code,
		DeliveryFee:      p.deliveryMethod.Fee,
		DeliveryOfficeID: p.deliveryOfficeID,
		PaymentMethod:    p.paymentMethod,
		Payment: &OrderPaymentRecord{
			Provider:        paymentProviderRevolut,
			ProviderOrderID: paymentOrder.ID,
			Status:          paymentStatusPending,
			Amount:          p.total,
		},
		ReservationID:  reservationID,
		Status:         orderStatusPendingPayment,
		Total:          p.total,
		DiscountCode:   p.discountCode,
		DiscountAmount: p.discountAmount,
		Items:          p.items,
	})
	if err != nil {
		_ = s.inventory.Release(ctx, p.reservationID, &p.userID)
		return PlaceOrderResult{}, err
	}

	// Reserve the discount code use now — an abandoned payment consuming a use
	// is an acceptable, rare trade-off vs. the complexity of releasing it on the
	// failure webhook. Clearing the cart here (we still have the owner) means a
	// guest's cart is emptied against the reserved stock, matching the on-hold
	// intent even though the money isn't captured yet.
	if p.discountCodeID != uuid.Nil {
		if err := s.discounts.UseCode(ctx, p.discountCodeID); err != nil {
			s.logger.Error("failed to increment discount code use", "error", err, "code_id", p.discountCodeID)
		}
	}
	_ = s.cart.ClearCart(ctx, owner)

	return PlaceOrderResult{PaymentRequired: &PaymentInitiation{
		OrderID:           result.ID,
		OrderNumber:       p.orderNumber,
		RevolutOrderID:    paymentOrder.ID,
		RevolutOrderToken: paymentOrder.Token,
		Amount:            p.total,
		PaymentMethod:     p.paymentMethod,
		Status:            orderStatusPendingPayment,
	}}, nil
}

// PaymentStatus returns the current status of an order by its number, for the
// storefront's post-payment confirmation poll.
func (s *Service) PaymentStatus(ctx context.Context, orderNumber string) (string, error) {
	return s.orders.GetStatusByNumber(ctx, orderNumber)
}

// FinalizePaidOrder settles a card order once its payment is confirmed. Called
// from the Revolut webhook (Phase 3). Idempotent: it re-fetches the order state
// and returns nil if the order is already settled or not awaiting payment, and
// re-confirms the payment authoritatively with the gateway before committing
// stock, marking the order paid, and booking the shipment + invoice.
func (s *Service) FinalizePaidOrder(ctx context.Context, providerOrderID string) error {
	ord, err := s.orders.FindByProviderOrderID(ctx, providerOrderID)
	if err != nil {
		return err
	}
	if ord.Status != orderStatusPendingPayment {
		return nil // already settled/failed — nothing to do
	}

	paymentOrder, err := s.payments.GetOrder(ctx, providerOrderID)
	if err != nil {
		return err
	}
	if paymentOrder.State != PaymentStateCompleted {
		return nil // not paid yet (e.g. an authorised-only event) — wait for the completion webhook
	}
	if paymentOrder.AmountMinor != ord.Total.AmountMinor || paymentOrder.Currency != ord.Total.Currency {
		s.logger.Error("revolut payment amount mismatch", "order_number", ord.OrderNumber,
			"expected_minor", ord.Total.AmountMinor, "got_minor", paymentOrder.AmountMinor)
		return domain.ErrPaymentAmountMismatch
	}

	if ord.ReservationID != nil {
		if err := s.inventory.Commit(ctx, *ord.ReservationID, &ord.UserID); err != nil {
			return err
		}
	}
	if err := s.orders.MarkPaid(ctx, ord.ID, paymentOrder.ID, paymentOrder.AmountMinor); err != nil {
		return err
	}

	if deliveryMethod, ok := domain.FindDeliveryMethod(ord.DeliveryMethod); ok {
		s.createShipment(ctx, OrderResult{ID: ord.ID, OrderNumber: ord.OrderNumber}, deliveryMethod,
			addressFromOrder(ord.ShippingAddress), ord.ContactName, ord.ContactPhone, ord.ContactEmail,
			ord.DeliveryOfficeID, ord.PaymentMethod, ord.Total)
	}
	if s.invoices != nil {
		if err := s.invoices.GenerateForOrder(ctx, ord.ID); err != nil {
			s.logger.Error("failed to generate invoice for order", "error", err, "order_id", ord.ID)
		}
	}
	return nil
}

// FailPayment releases the reservation and marks a card order payment_failed
// after a declined/cancelled/abandoned payment. Idempotent.
func (s *Service) FailPayment(ctx context.Context, providerOrderID, reason string) error {
	ord, err := s.orders.FindByProviderOrderID(ctx, providerOrderID)
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			return nil
		}
		return err
	}
	if ord.Status != orderStatusPendingPayment {
		return nil
	}
	if ord.ReservationID != nil {
		_ = s.inventory.Release(ctx, *ord.ReservationID, &ord.UserID)
	}
	return s.orders.MarkPaymentFailed(ctx, ord.ID, reason)
}

// RefundOrder issues a (possibly partial) refund against a settled card order
// via Revolut and records it. The order's rolled-up status is advanced to
// refunded / partially_refunded only when the refund completes.
func (s *Service) RefundOrder(ctx context.Context, orderID uuid.UUID, amountMinor int64, reason string, adminID *uuid.UUID) error {
	pc, err := s.orders.GetPaymentContext(ctx, orderID)
	if err != nil {
		return err
	}
	if pc.ProviderOrderID == "" {
		return domain.ErrRefundNotAllowed
	}
	switch pc.OrderStatus {
	case orderStatusPaid, "shipped", "delivered", "partially_refunded":
	default:
		return domain.ErrRefundNotAllowed
	}
	refundable := pc.CapturedMinor - pc.RefundedMinor
	if amountMinor <= 0 || amountMinor > refundable {
		return domain.ErrRefundAmountInvalid
	}

	res, err := s.payments.Refund(ctx, RefundInput{
		ProviderOrderID: pc.ProviderOrderID,
		Amount:          money.Money{AmountMinor: amountMinor, Currency: pc.Currency},
		Reason:          reason,
	})
	if err != nil {
		s.logger.Error("revolut refund failed", "error", err, "order_id", orderID)
		return domain.ErrRefundFailed
	}

	state := res.State
	if state == "" {
		state = paymentStatusPending
	}
	orderStatus := ""
	if state == "completed" {
		if pc.RefundedMinor+amountMinor >= pc.CapturedMinor {
			orderStatus = "refunded"
		} else {
			orderStatus = "partially_refunded"
		}
	}
	return s.orders.RecordRefund(ctx, RecordRefundInput{
		OrderID:          orderID,
		ProviderRefundID: res.ID,
		Amount:           money.Money{AmountMinor: amountMinor, Currency: pc.Currency},
		Reason:           reason,
		State:            state,
		CreatedBy:        adminID,
		OrderStatus:      orderStatus,
	})
}

// createShipment books the real Speedy shipment for a just-placed order and
// writes the tracking info back onto it. Failures are logged, never
// returned — a customer's order stays placed even if fulfillment can't be
// booked immediately; an admin can retry by other means later.
func (s *Service) createShipment(ctx context.Context, order OrderResult, deliveryMethod domain.DeliveryMethod, shippingAddr domain.Address, contactName, contactPhone, contactEmail, officeID, paymentMethod string, total money.Money) {
	shipment, err := s.fulfillment.CreateShipment(ctx, CreateShipmentInput{
		Provider:       domain.ProviderFor(deliveryMethod.Code),
		DeliveryMethod: deliveryMethod.Code,
		OfficeID:       officeID,
		ContactName:    contactName,
		Phone:          contactPhone,
		Email:          contactEmail,
		Address:        toOrderAddress(shippingAddr),
		RequireCOD:     paymentMethod != domain.PaymentMethodCardOnline,
		CODAmount:      total,
		Ref1:           order.OrderNumber,
	})
	if err != nil {
		s.logger.Error("failed to create logistics shipment for order", "error", err, "order_id", order.ID, "order_number", order.OrderNumber)
		return
	}

	if err := s.orders.SetShipmentInfo(ctx, order.ID, domain.ProviderFor(deliveryMethod.Code), shipment.ParcelID, shipment.ShipmentID, "created"); err != nil {
		s.logger.Error("failed to record shipment info on order", "error", err, "order_id", order.ID)
	}
}
