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
	"github.com/adotomov/fashion-store/apps/api/internal/platform/metrics"
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

// ReserveCheckoutSession acquires (or extends) the stock hold for a shopper's
// checkout, called when they enter checkout. Holding it for the whole session —
// rather than only during payment — means a shopper who's filling in details
// won't be beaten to the last unit at the payment step, and switching payment
// methods never re-touches stock. A second shopper for the same last unit is
// told out-of-stock up front (409) instead of mid-checkout. Reuses a live hold
// (extending its expiry); otherwise reserves fresh, first dropping any lapsed
// hold so it isn't orphaned. Returns the (new) expiry.
func (s *Service) ReserveCheckoutSession(ctx context.Context, owner CartOwner) (time.Time, error) {
	cartSnap, err := s.cart.GetCart(ctx, owner)
	if err != nil {
		return time.Time{}, err
	}
	if len(cartSnap.Lines) == 0 {
		return time.Time{}, domain.ErrCartEmpty
	}

	now := time.Now()
	expiresAt := now.Add(DefaultAbandonedPaymentTTL)

	existing, err := s.cart.GetReservation(ctx, owner)
	if err != nil {
		return time.Time{}, err
	}
	if existing != nil {
		if existing.ExpiresAt.After(now) {
			// Live hold — extend and reuse. (The cart isn't editable during
			// checkout; a cart edit releases the hold via ReleaseCheckoutSession,
			// so a live hold still matches the cart.)
			if err := s.cart.SetReservation(ctx, owner, existing.ReservationID, expiresAt); err != nil {
				return time.Time{}, err
			}
			return expiresAt, nil
		}
		// Lapsed hold — release it before acquiring a fresh one. A failed release
		// orphans the old inventory reservation (nothing points to it once the
		// cart columns are overwritten below), so surface it for troubleshooting.
		if err := s.inventory.Release(ctx, existing.ReservationID, owner.UserID); err != nil {
			s.logger.WarnContext(ctx, "failed to release lapsed checkout reservation", "error", err, "reservation_id", existing.ReservationID)
		}
	}

	reserveLines := make([]ReserveLine, 0, len(cartSnap.Lines))
	for _, line := range cartSnap.Lines {
		reserveLines = append(reserveLines, ReserveLine{VariantID: line.VariantID, Quantity: line.Quantity})
	}
	reservationID, err := s.inventory.Reserve(ctx, reserveLines, owner.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrInsufficientStock) {
			metrics.ReservationConflict(ctx)
		}
		return time.Time{}, err // ErrInsufficientStock → 409
	}
	if err := s.cart.SetReservation(ctx, owner, reservationID, expiresAt); err != nil {
		_ = s.inventory.Release(ctx, reservationID, owner.UserID)
		return time.Time{}, err
	}
	return expiresAt, nil
}

// ReleaseCheckoutSession drops a shopper's checkout hold, returning the stock —
// called best-effort when they explicitly leave checkout (or edit their cart).
// Silent abandonment is caught by the sweeper instead. No hold is a no-op.
func (s *Service) ReleaseCheckoutSession(ctx context.Context, owner CartOwner) error {
	existing, err := s.cart.GetReservation(ctx, owner)
	if err != nil {
		return err
	}
	if existing == nil {
		return nil
	}
	_ = s.inventory.Release(ctx, existing.ReservationID, owner.UserID)
	return s.cart.ClearReservation(ctx, owner)
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

	// Reuse the checkout-session stock hold if one is live (acquired when the
	// shopper entered checkout and reused across payment-method switches). With a
	// hold in place the cart's AvailableQuantity already nets out our own
	// reservation, so the per-line availability check is skipped — it's the hold,
	// not this snapshot, that guarantees the stock.
	sessionRes, err := s.cart.GetReservation(ctx, owner)
	if err != nil {
		return PlaceOrderResult{}, err
	}
	hasSessionHold := sessionRes != nil && sessionRes.ExpiresAt.After(time.Now())

	var subtotal int64
	currency := deliveryMethod.Fee.Currency
	reserveLines := make([]ReserveLine, 0, len(cartSnap.Lines))
	orderItems := make([]CreateOrderItemInput, 0, len(cartSnap.Lines))
	for _, line := range cartSnap.Lines {
		if !hasSessionHold && line.Quantity > line.AvailableQuantity {
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

	var reservationID uuid.UUID
	sessionOwned := hasSessionHold
	if hasSessionHold {
		reservationID = sessionRes.ReservationID
	} else {
		// No live checkout hold (COD placed straight from the cart, or the hold
		// lapsed) — reserve inline now, dropping a lapsed hold first so it isn't
		// left orphaned.
		if sessionRes != nil {
			if err := s.inventory.Release(ctx, sessionRes.ReservationID, owner.UserID); err != nil {
				s.logger.WarnContext(ctx, "failed to release lapsed checkout reservation", "error", err, "reservation_id", sessionRes.ReservationID)
			}
			_ = s.cart.ClearReservation(ctx, owner)
		}
		reservationID, err = s.inventory.Reserve(ctx, reserveLines, &userID)
		if err != nil {
			return PlaceOrderResult{}, err
		}
		// For card payments, record the inline reservation as the checkout-session
		// hold so its lifecycle is owned by the cart (retryable across payment
		// switches, reclaimed by the sweeper) rather than this single pending order.
		if domain.RequiresUpfrontPayment(input.PaymentMethod) {
			if err := s.cart.SetReservation(ctx, owner, reservationID, time.Now().Add(DefaultAbandonedPaymentTTL)); err != nil {
				_ = s.inventory.Release(ctx, reservationID, &userID)
				return PlaceOrderResult{}, err
			}
			sessionOwned = true
		}
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
			sessionOwned:     sessionOwned,
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
		_ = s.cart.ClearReservation(ctx, owner)
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
			s.logger.ErrorContext(ctx, "failed to increment discount code use", "error", err, "code_id", discountCodeID)
		}
	}

	metrics.OrderPlaced(ctx, input.PaymentMethod)

	// Stock is committed and the order placed: drop the cart and its now-consumed
	// checkout hold.
	_ = s.cart.ClearReservation(ctx, owner)
	_ = s.cart.ClearCart(ctx, owner)

	s.createShipment(ctx, result, deliveryMethod, shippingAddr, contactName, contactPhone, contactEmail, input.DeliveryOfficeID, input.PaymentMethod, total)

	// Generate the invoice for every successfully placed order, regardless of
	// payment or delivery method. Runs after shipment booking so COD/EasyBox
	// invoices capture the assigned courier. GenerateForOrder is idempotent, so
	// the orders module's delivered-status trigger stays a harmless retry.
	if s.invoices != nil {
		if err := s.invoices.GenerateForOrder(ctx, result.ID); err != nil {
			s.logger.ErrorContext(ctx, "failed to generate invoice for order", "error", err, "order_id", result.ID)
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
	orderNumber   string
	userID        uuid.UUID
	reservationID uuid.UUID
	// sessionOwned is true when reservationID is the checkout-session hold (owned
	// by the cart, not this order) — in which case a failure here must NOT release
	// it, so a retry can reuse the same hold.
	sessionOwned     bool
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
		if !p.sessionOwned {
			_ = s.inventory.Release(ctx, p.reservationID, &p.userID)
		}
		s.logger.ErrorContext(ctx, "failed to create revolut order", "error", err, "order_number", p.orderNumber)
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
		CartGuestToken: owner.GuestToken,
		Status:         orderStatusPendingPayment,
		Total:          p.total,
		DiscountCode:   p.discountCode,
		DiscountAmount: p.discountAmount,
		Items:          p.items,
	})
	if err != nil {
		if !p.sessionOwned {
			_ = s.inventory.Release(ctx, p.reservationID, &p.userID)
		}
		return PlaceOrderResult{}, err
	}

	// Reserve the discount code use now — an abandoned payment consuming a use
	// is an acceptable, rare trade-off vs. the complexity of releasing it on the
	// failure webhook.
	if p.discountCodeID != uuid.Nil {
		if err := s.discounts.UseCode(ctx, p.discountCodeID); err != nil {
			s.logger.ErrorContext(ctx, "failed to increment discount code use", "error", err, "code_id", p.discountCodeID)
		}
	}

	metrics.PaymentInitiated(ctx)

	// NB: the cart is deliberately NOT cleared here. It's cleared only once the
	// payment settles (FinalizePaidOrder), so that a customer who abandons the
	// widget, fails payment, or cancels keeps their cart. The guest cart token is
	// stored on the order (CartGuestToken) so the webhook can clear the right cart.

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
		s.logger.ErrorContext(ctx, "revolut payment amount mismatch", "order_number", ord.OrderNumber,
			"expected_minor", ord.Total.AmountMinor, "got_minor", paymentOrder.AmountMinor)
		metrics.PaymentFailed(ctx, "amount_mismatch")
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

	metrics.PaymentSucceeded(ctx)
	metrics.OrderPlaced(ctx, ord.PaymentMethod)

	// Payment confirmed: clear the cart now (deferred from initiation so an
	// abandoned/failed/cancelled payment keeps it). A guest's cart is keyed by
	// the token captured on the order; a signed-in user's by their id. Best
	// effort — a stale cart must never fail an otherwise-settled order.
	clearOwner := CartOwner{UserID: &ord.UserID}
	if ord.CartGuestToken != nil {
		clearOwner = CartOwner{GuestToken: ord.CartGuestToken}
	}
	if err := s.cart.ClearCart(ctx, clearOwner); err != nil {
		s.logger.ErrorContext(ctx, "failed to clear cart after payment", "error", err, "order_id", ord.ID)
	}
	// The checkout-session hold has now been committed above; drop its columns
	// from the cart so the sweeper doesn't later try to reclaim it.
	_ = s.cart.ClearReservation(ctx, clearOwner)

	if deliveryMethod, ok := domain.FindDeliveryMethod(ord.DeliveryMethod); ok {
		s.createShipment(ctx, OrderResult{ID: ord.ID, OrderNumber: ord.OrderNumber}, deliveryMethod,
			addressFromOrder(ord.ShippingAddress), ord.ContactName, ord.ContactPhone, ord.ContactEmail,
			ord.DeliveryOfficeID, ord.PaymentMethod, ord.Total)
	}
	if s.invoices != nil {
		if err := s.invoices.GenerateForOrder(ctx, ord.ID); err != nil {
			s.logger.ErrorContext(ctx, "failed to generate invoice for order", "error", err, "order_id", ord.ID)
		}
	}
	return nil
}

// FailPayment marks a card order payment_failed after a declined/cancelled/
// abandoned payment. It deliberately does NOT release stock: the reservation is
// the checkout-session hold owned by the cart, so a failed/cancelled attempt
// keeps it for a retry with another method. The hold is reclaimed by the
// checkout-reservation sweeper on abandonment, or committed on settlement.
// Idempotent.
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
	if err := s.orders.MarkPaymentFailed(ctx, ord.ID, reason); err != nil {
		return err
	}
	metrics.PaymentFailed(ctx, reason)
	return nil
}

// CancelPendingPayment is the customer-initiated counterpart to the abandoned-
// payment sweeper: when a shopper backs out of the card widget to pick another
// method, release the held stock and mark the order payment_failed right away
// (rather than waiting ~30 min for the sweeper). The cart was never cleared for
// a pending-payment order, so backing out leaves it intact. Authorised by proving
// knowledge of the order's (unguessable) provider order id — no session required,
// so it works for guests too. Idempotent and safe to call on an already-settled
// or unknown order (no-op).
func (s *Service) CancelPendingPayment(ctx context.Context, orderNumber, providerOrderID string) error {
	ord, err := s.orders.FindByProviderOrderID(ctx, providerOrderID)
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			return nil
		}
		return err
	}
	// The provider order id must belong to the claimed order number — this is the
	// capability check that keeps one customer from cancelling another's order.
	if ord.OrderNumber != orderNumber {
		return domain.ErrOrderNotFound
	}
	return s.FailPayment(ctx, providerOrderID, "cancelled_by_customer")
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
		s.logger.ErrorContext(ctx, "revolut refund failed", "error", err, "order_id", orderID)
		metrics.RefundRecorded(ctx, "failed")
		return domain.ErrRefundFailed
	}

	state := res.State
	if state == "" {
		state = paymentStatusPending
	}
	metrics.RefundRecorded(ctx, state)
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
