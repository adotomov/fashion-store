package application

import (
	"context"
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
	logger      *slog.Logger
}

func NewService(cart CartGateway, inventory InventoryGateway, users UserGateway, orders OrderGateway, payments PaymentGateway, fulfillment FulfillmentGateway, discounts DiscountGateway, invoices InvoiceGateway, logger *slog.Logger) *Service {
	return &Service{cart: cart, inventory: inventory, users: users, orders: orders, payments: payments, fulfillment: fulfillment, discounts: discounts, invoices: invoices, logger: logger}
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

// PlaceOrder orchestrates the whole checkout: validate the cart and chosen
// methods, reserve stock for every line (so a card charge can't oversell
// while it's in flight), charge the mocked Revolut gateway if the payment
// method requires it upfront, then either commit the reservation and
// create the order or release the reservation and fail. Reservation is
// always resolved (committed or released) before PlaceOrder returns —
// there's no in-between state left for a background job to clean up.
func (s *Service) PlaceOrder(ctx context.Context, owner CartOwner, principalUserID *uuid.UUID, input PlaceOrderInput) (OrderResult, error) {
	cartSnap, err := s.cart.GetCart(ctx, owner)
	if err != nil {
		return OrderResult{}, err
	}
	if len(cartSnap.Lines) == 0 {
		return OrderResult{}, domain.ErrCartEmpty
	}

	deliveryMethod, ok := domain.FindDeliveryMethod(input.DeliveryMethod)
	if !ok {
		return OrderResult{}, domain.ErrInvalidDeliveryMethod
	}
	if !domain.ValidPaymentMethod(input.PaymentMethod) {
		return OrderResult{}, domain.ErrInvalidPaymentMethod
	}
	if !s.fulfillment.IsProviderEnabled(ctx, domain.ProviderFor(deliveryMethod.Code)) {
		return OrderResult{}, domain.ErrDeliveryMethodUnavailable
	}
	if deliveryMethod.Code == domain.DeliveryMethodEasyBox && input.DeliveryOfficeID == "" {
		return OrderResult{}, domain.ErrOfficeRequired
	}

	shippingAddr := input.ShippingAddress.toDomain()
	if err := shippingAddr.Validate(); err != nil {
		return OrderResult{}, err
	}
	billingAddr := input.BillingAddress.toDomain()
	if err := billingAddr.Validate(); err != nil {
		return OrderResult{}, err
	}

	var userID uuid.UUID
	contactName, contactEmail, contactPhone := input.Contact.FullName, input.Contact.Email, input.Contact.Phone
	if principalUserID != nil {
		userID = *principalUserID
	} else {
		contact := domain.Contact{FullName: contactName, Email: contactEmail, Phone: contactPhone}
		if err := contact.Validate(); err != nil {
			return OrderResult{}, err
		}
		id, err := s.users.EnsureUser(ctx, contact.Email, contact.FullName)
		if err != nil {
			return OrderResult{}, err
		}
		userID = id
	}

	var subtotal int64
	currency := deliveryMethod.Fee.Currency
	reserveLines := make([]ReserveLine, 0, len(cartSnap.Lines))
	orderItems := make([]CreateOrderItemInput, 0, len(cartSnap.Lines))
	for _, line := range cartSnap.Lines {
		if line.Quantity > line.AvailableQuantity {
			return OrderResult{}, domain.ErrInsufficientStock
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
			return OrderResult{}, domain.ErrInvalidDiscountCode
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
		return OrderResult{}, err
	}

	status := "pending"
	var paymentRecord *OrderPaymentRecord
	if domain.RequiresUpfrontPayment(input.PaymentMethod) {
		chargeResult, chargeErr := s.payments.Charge(ctx, ChargeInput{
			Amount:   total,
			OrderRef: orderNumber,
			Card:     input.Card,
		})
		if chargeErr != nil || !chargeResult.Succeeded {
			_ = s.inventory.Release(ctx, reservationID, &userID)
			return OrderResult{}, domain.ErrPaymentFailed
		}
		paymentRecord = &OrderPaymentRecord{
			Provider:          "revolut_mock",
			ProviderReference: chargeResult.ProviderReference,
			Status:            "succeeded",
			Amount:            total,
		}
		status = "paid"
	}

	if err := s.inventory.Commit(ctx, reservationID, &userID); err != nil {
		_ = s.inventory.Release(ctx, reservationID, &userID)
		return OrderResult{}, err
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
		Payment:          paymentRecord,
		ReservationID:    reservationID,
		Status:           status,
		Total:            total,
		DiscountCode:     discountCodeStr,
		DiscountAmount:   discountAmount,
		Items:            orderItems,
	})
	if err != nil {
		return OrderResult{}, err
	}

	// Increment discount code use count after successful order creation.
	if discountCodeID != uuid.Nil {
		if err := s.discounts.UseCode(ctx, discountCodeID); err != nil {
			s.logger.Error("failed to increment discount code use", "error", err, "code_id", discountCodeID)
		}
	}

	// Generate invoice for card_online orders that are immediately paid.
	// COD/EasyBox invoices are generated when the order reaches "delivered" status.
	if status == "paid" && s.invoices != nil {
		if err := s.invoices.GenerateForOrder(ctx, result.ID); err != nil {
			s.logger.Error("failed to generate invoice for order", "error", err, "order_id", result.ID)
		}
	}

	_ = s.cart.ClearCart(ctx, owner)

	s.createShipment(ctx, result, deliveryMethod, shippingAddr, contactName, contactPhone, contactEmail, input.DeliveryOfficeID, input.PaymentMethod, total)

	return result, nil
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
