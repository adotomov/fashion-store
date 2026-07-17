package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/app"
	adminapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/admin/application"
	admininfra "github.com/adotomov/fashion-store/apps/api/internal/modules/admin/infrastructure"
	adminhttp "github.com/adotomov/fashion-store/apps/api/internal/modules/admin/transport/http"
	authapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/auth/application"
	authinfra "github.com/adotomov/fashion-store/apps/api/internal/modules/auth/infrastructure"
	authhttp "github.com/adotomov/fashion-store/apps/api/internal/modules/auth/transport/http"
	cartapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/cart/application"
	cartinfra "github.com/adotomov/fashion-store/apps/api/internal/modules/cart/infrastructure"
	carthttp "github.com/adotomov/fashion-store/apps/api/internal/modules/cart/transport/http"
	catalogapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/application"
	cataloginfra "github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/infrastructure"
	cataloghttp "github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/transport/http"
	checkoutapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/checkout/application"
	checkoutdomain "github.com/adotomov/fashion-store/apps/api/internal/modules/checkout/domain"
	checkoutinfra "github.com/adotomov/fashion-store/apps/api/internal/modules/checkout/infrastructure"
	checkouthttp "github.com/adotomov/fashion-store/apps/api/internal/modules/checkout/transport/http"
	fulfillmentapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/fulfillment/application"
	fulfillmentinfra "github.com/adotomov/fashion-store/apps/api/internal/modules/fulfillment/infrastructure"
	fulfillmenthttp "github.com/adotomov/fashion-store/apps/api/internal/modules/fulfillment/transport/http"
	i18napplication "github.com/adotomov/fashion-store/apps/api/internal/modules/i18n/application"
	i18ninfra "github.com/adotomov/fashion-store/apps/api/internal/modules/i18n/infrastructure"
	i18nhttp "github.com/adotomov/fashion-store/apps/api/internal/modules/i18n/transport/http"
	inventoryapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/inventory/application"
	inventorydomain "github.com/adotomov/fashion-store/apps/api/internal/modules/inventory/domain"
	inventoryinfra "github.com/adotomov/fashion-store/apps/api/internal/modules/inventory/infrastructure"
	inventoryhttp "github.com/adotomov/fashion-store/apps/api/internal/modules/inventory/transport/http"
	invoicingapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/invoicing/application"
	invoicinginfra "github.com/adotomov/fashion-store/apps/api/internal/modules/invoicing/infrastructure"
	invoicinghttp "github.com/adotomov/fashion-store/apps/api/internal/modules/invoicing/transport/http"
	ordersapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/orders/application"
	ordersdomain "github.com/adotomov/fashion-store/apps/api/internal/modules/orders/domain"
	ordersinfra "github.com/adotomov/fashion-store/apps/api/internal/modules/orders/infrastructure"
	ordershttp "github.com/adotomov/fashion-store/apps/api/internal/modules/orders/transport/http"
	paymentsapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/payments/application"
	paymentsinfra "github.com/adotomov/fashion-store/apps/api/internal/modules/payments/infrastructure"
	paymentshttp "github.com/adotomov/fashion-store/apps/api/internal/modules/payments/transport/http"
	promotionsapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/promotions/application"
	promotionsinfra "github.com/adotomov/fashion-store/apps/api/internal/modules/promotions/infrastructure"
	promotionshttp "github.com/adotomov/fashion-store/apps/api/internal/modules/promotions/transport/http"
	usersapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/users/application"
	usersinfra "github.com/adotomov/fashion-store/apps/api/internal/modules/users/infrastructure"
	usershttp "github.com/adotomov/fashion-store/apps/api/internal/modules/users/transport/http"
	wishlistapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/wishlist/application"
	wishlistinfra "github.com/adotomov/fashion-store/apps/api/internal/modules/wishlist/infrastructure"
	wishlisthttp "github.com/adotomov/fashion-store/apps/api/internal/modules/wishlist/transport/http"
	"github.com/adotomov/fashion-store/apps/api/internal/platform/googleauth"
	"github.com/adotomov/fashion-store/apps/api/internal/platform/storage"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

// userProvisionerAdapter implements auth's UserProvisioner port on top of
// the users module's application service, keeping the two modules decoupled
// (auth never imports users' infrastructure or domain types directly).
type userProvisionerAdapter struct {
	users *usersapplication.Service
}

func (a *userProvisionerAdapter) EnsureUser(ctx context.Context, input authapplication.EnsureUserInput) (authapplication.UserRef, error) {
	user, err := a.users.EnsureUser(ctx, usersapplication.CreateUserInput{
		Email:    input.Email,
		FullName: input.FullName,
	})
	if err != nil {
		return authapplication.UserRef{}, err
	}
	return authapplication.UserRef{ID: user.ID}, nil
}

func (a *userProvisionerAdapter) GetRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return a.users.GetRoles(ctx, userID)
}

// checkoutCartGatewayAdapter implements checkout's CartGateway port on top
// of the cart module's application service.
type checkoutCartGatewayAdapter struct {
	cart *cartapplication.Service
}

func (a *checkoutCartGatewayAdapter) GetCart(ctx context.Context, owner checkoutapplication.CartOwner) (checkoutapplication.CartSnapshot, error) {
	cart, err := a.cart.GetCart(ctx, cartapplication.CartOwner{UserID: owner.UserID, GuestToken: owner.GuestToken})
	if err != nil {
		return checkoutapplication.CartSnapshot{}, err
	}
	lines := make([]checkoutapplication.CartLine, 0, len(cart.Items))
	for _, item := range cart.Items {
		lines = append(lines, checkoutapplication.CartLine{
			VariantID:         item.VariantID,
			ProductID:         item.ProductID,
			ProductName:       item.ProductName,
			VariantLabel:      item.VariantLabel,
			Quantity:          item.Quantity,
			UnitPrice:         item.UnitPrice,
			AvailableQuantity: item.AvailableQuantity,
		})
	}
	return checkoutapplication.CartSnapshot{ID: cart.ID, Lines: lines}, nil
}

func (a *checkoutCartGatewayAdapter) ClearCart(ctx context.Context, owner checkoutapplication.CartOwner) error {
	return a.cart.ClearCart(ctx, cartapplication.CartOwner{UserID: owner.UserID, GuestToken: owner.GuestToken})
}

// checkoutInventoryGatewayAdapter implements checkout's InventoryGateway
// port on top of the inventory module's application service, translating
// the insufficient-stock error into checkout's own domain error so the
// checkout HTTP layer never needs to import inventory's domain package.
type checkoutInventoryGatewayAdapter struct {
	inventory *inventoryapplication.Service
}

func (a *checkoutInventoryGatewayAdapter) Reserve(ctx context.Context, lines []checkoutapplication.ReserveLine, createdBy *uuid.UUID) (uuid.UUID, error) {
	invLines := make([]inventoryapplication.ReserveLine, 0, len(lines))
	for _, l := range lines {
		invLines = append(invLines, inventoryapplication.ReserveLine{VariantID: l.VariantID, Quantity: l.Quantity})
	}
	reservation, err := a.inventory.ReserveForVariants(ctx, invLines, createdBy)
	if err != nil {
		if errors.Is(err, inventorydomain.ErrInsufficientStock) {
			return uuid.Nil, checkoutdomain.ErrInsufficientStock
		}
		return uuid.Nil, err
	}
	return reservation.ID, nil
}

func (a *checkoutInventoryGatewayAdapter) Commit(ctx context.Context, reservationID uuid.UUID, createdBy *uuid.UUID) error {
	return a.inventory.CommitReservation(ctx, reservationID, createdBy)
}

func (a *checkoutInventoryGatewayAdapter) Release(ctx context.Context, reservationID uuid.UUID, createdBy *uuid.UUID) error {
	return a.inventory.ReleaseReservation(ctx, reservationID, createdBy)
}

// checkoutUserGatewayAdapter implements checkout's UserGateway port,
// reusing the same find-or-create-by-email path the auth module uses to
// provision accounts on first Google login — a guest checkout user who
// later signs in with the same email picks up their order history.
type checkoutUserGatewayAdapter struct {
	users *usersapplication.Service
}

func (a *checkoutUserGatewayAdapter) EnsureUser(ctx context.Context, email, fullName string) (uuid.UUID, error) {
	user, err := a.users.EnsureUser(ctx, usersapplication.CreateUserInput{Email: email, FullName: fullName})
	if err != nil {
		return uuid.Nil, err
	}
	return user.ID, nil
}

// checkoutOrderGatewayAdapter implements checkout's OrderGateway port on
// top of the orders module's application service.
type checkoutOrderGatewayAdapter struct {
	orders *ordersapplication.Service
}

func (a *checkoutOrderGatewayAdapter) CreateOrder(ctx context.Context, input checkoutapplication.CreateOrderInput) (checkoutapplication.OrderResult, error) {
	var payment *ordersapplication.CreateOrderPaymentInput
	if input.Payment != nil {
		payment = &ordersapplication.CreateOrderPaymentInput{
			Provider:          input.Payment.Provider,
			ProviderReference: input.Payment.ProviderReference,
			Status:            input.Payment.Status,
			Amount:            input.Payment.Amount,
		}
	}

	items := make([]ordersapplication.CreateOrderItemInput, 0, len(input.Items))
	for _, item := range input.Items {
		var productID *uuid.UUID
		if item.ProductID != uuid.Nil {
			productID = &item.ProductID
		}
		items = append(items, ordersapplication.CreateOrderItemInput{
			ProductID:    productID,
			ProductName:  item.ProductName,
			VariantLabel: item.VariantLabel,
			Quantity:     item.Quantity,
			UnitPrice:    item.UnitPrice,
		})
	}

	reservationID := input.ReservationID
	order, err := a.orders.CreateOrder(ctx, input.UserID, ordersapplication.CreateOrderInput{
		OrderNumber:  input.OrderNumber,
		Status:       input.Status,
		Total:        input.Total,
		PlacedAt:     time.Now(),
		ContactName:  input.ContactName,
		ContactEmail: input.ContactEmail,
		ContactPhone: input.ContactPhone,
		ShippingAddress: ordersdomain.OrderAddress{
			RecipientName: input.ShippingAddress.RecipientName, Phone: input.ShippingAddress.Phone,
			Line1: input.ShippingAddress.Line1, Line2: input.ShippingAddress.Line2,
			City: input.ShippingAddress.City, Region: input.ShippingAddress.Region,
			PostalCode: input.ShippingAddress.PostalCode, CountryCode: input.ShippingAddress.CountryCode,
		},
		BillingAddress: ordersdomain.OrderAddress{
			RecipientName: input.BillingAddress.RecipientName, Phone: input.BillingAddress.Phone,
			Line1: input.BillingAddress.Line1, Line2: input.BillingAddress.Line2,
			City: input.BillingAddress.City, Region: input.BillingAddress.Region,
			PostalCode: input.BillingAddress.PostalCode, CountryCode: input.BillingAddress.CountryCode,
		},
		DeliveryMethod: input.DeliveryMethod,
		DeliveryFee:    input.DeliveryFee,
		PaymentMethod:  input.PaymentMethod,
		Payment:        payment,
		ReservationID:  &reservationID,
		CartGuestToken: input.CartGuestToken,
		Items:          items,
	})
	if err != nil {
		return checkoutapplication.OrderResult{}, err
	}

	resultItems := make([]checkoutapplication.OrderResultItem, 0, len(order.Items))
	for _, item := range order.Items {
		resultItems = append(resultItems, checkoutapplication.OrderResultItem{
			ProductName:  item.ProductName,
			VariantLabel: item.VariantLabel,
			Quantity:     item.Quantity,
			UnitPrice:    item.UnitPrice,
		})
	}

	return checkoutapplication.OrderResult{
		ID:             order.ID,
		OrderNumber:    order.OrderNumber,
		Status:         string(order.Status),
		Total:          order.Total,
		DeliveryMethod: order.DeliveryMethod,
		DeliveryFee:    order.DeliveryFee,
		PaymentMethod:  order.PaymentMethod,
		PlacedAt:       order.PlacedAt,
		Items:          resultItems,
	}, nil
}

func (a *checkoutOrderGatewayAdapter) SetShipmentInfo(ctx context.Context, orderID uuid.UUID, carrier, trackingNumber, shipmentID, status string) error {
	_, err := a.orders.UpdateFulfillment(ctx, orderID, ordersapplication.UpdateFulfillmentInput{
		Carrier:        &carrier,
		TrackingNumber: &trackingNumber,
		ShipmentID:     &shipmentID,
		ShipmentStatus: &status,
	})
	return err
}

func (a *checkoutOrderGatewayAdapter) FindByProviderOrderID(ctx context.Context, providerOrderID string) (checkoutapplication.OrderForFinalize, error) {
	order, err := a.orders.FindByProviderOrderID(ctx, providerOrderID)
	if err != nil {
		if errors.Is(err, ordersdomain.ErrOrderNotFound) {
			return checkoutapplication.OrderForFinalize{}, checkoutdomain.ErrOrderNotFound
		}
		return checkoutapplication.OrderForFinalize{}, err
	}
	var officeID string
	if order.DeliveryOfficeID != nil {
		officeID = *order.DeliveryOfficeID
	}
	return checkoutapplication.OrderForFinalize{
		ID:               order.ID,
		OrderNumber:      order.OrderNumber,
		Status:           string(order.Status),
		UserID:           order.UserID,
		CartGuestToken:   order.CartGuestToken,
		ReservationID:    order.ReservationID,
		DeliveryMethod:   order.DeliveryMethod,
		DeliveryOfficeID: officeID,
		PaymentMethod:    order.PaymentMethod,
		ContactName:      order.ContactName,
		ContactEmail:     order.ContactEmail,
		ContactPhone:     order.ContactPhone,
		ShippingAddress: checkoutapplication.OrderAddress{
			RecipientName: order.ShippingAddress.RecipientName, Phone: order.ShippingAddress.Phone,
			Line1: order.ShippingAddress.Line1, Line2: order.ShippingAddress.Line2,
			City: order.ShippingAddress.City, Region: order.ShippingAddress.Region,
			PostalCode: order.ShippingAddress.PostalCode, CountryCode: order.ShippingAddress.CountryCode,
		},
		Total: order.Total,
	}, nil
}

func (a *checkoutOrderGatewayAdapter) MarkPaid(ctx context.Context, orderID uuid.UUID, providerReference string, capturedMinor int64) error {
	return a.orders.MarkPaid(ctx, orderID, providerReference, capturedMinor)
}

func (a *checkoutOrderGatewayAdapter) MarkPaymentFailed(ctx context.Context, orderID uuid.UUID, reason string) error {
	return a.orders.MarkPaymentFailed(ctx, orderID, reason)
}

func (a *checkoutOrderGatewayAdapter) GetPaymentContext(ctx context.Context, orderID uuid.UUID) (checkoutapplication.OrderPaymentContext, error) {
	pc, err := a.orders.GetOrderPaymentContext(ctx, orderID)
	if err != nil {
		if errors.Is(err, ordersdomain.ErrOrderNotFound) {
			return checkoutapplication.OrderPaymentContext{}, checkoutdomain.ErrOrderNotFound
		}
		return checkoutapplication.OrderPaymentContext{}, err
	}
	return checkoutapplication.OrderPaymentContext{
		OrderStatus:     pc.Status,
		ProviderOrderID: pc.ProviderOrderID,
		CapturedMinor:   pc.CapturedMinor,
		RefundedMinor:   pc.RefundedMinor,
		Currency:        pc.Currency,
	}, nil
}

func (a *checkoutOrderGatewayAdapter) RecordRefund(ctx context.Context, input checkoutapplication.RecordRefundInput) error {
	return a.orders.RecordRefund(ctx, ordersapplication.RecordRefundInput{
		OrderID:          input.OrderID,
		ProviderRefundID: input.ProviderRefundID,
		AmountMinor:      input.Amount.AmountMinor,
		Currency:         input.Amount.Currency,
		Reason:           input.Reason,
		State:            input.State,
		CreatedBy:        input.CreatedBy,
		OrderStatus:      input.OrderStatus,
	})
}

func (a *checkoutOrderGatewayAdapter) GetStatusByNumber(ctx context.Context, orderNumber string) (string, error) {
	order, err := a.orders.FindByOrderNumber(ctx, orderNumber)
	if err != nil {
		if errors.Is(err, ordersdomain.ErrOrderNotFound) {
			return "", checkoutdomain.ErrOrderNotFound
		}
		return "", err
	}
	return string(order.Status), nil
}

func (a *checkoutOrderGatewayAdapter) ListPendingPaymentOlderThan(ctx context.Context, cutoff time.Time) ([]checkoutapplication.PendingPaymentRef, error) {
	refs, err := a.orders.ListPendingPaymentOlderThan(ctx, cutoff)
	if err != nil {
		return nil, err
	}
	out := make([]checkoutapplication.PendingPaymentRef, 0, len(refs))
	for _, ref := range refs {
		out = append(out, checkoutapplication.PendingPaymentRef{OrderID: ref.OrderID, ProviderOrderID: ref.ProviderOrderID})
	}
	return out, nil
}

// checkoutFulfillmentGatewayAdapter implements checkout's FulfillmentGateway
// port on top of the fulfillment module's application service.
type checkoutFulfillmentGatewayAdapter struct {
	fulfillment *fulfillmentapplication.Service
}

func (a *checkoutFulfillmentGatewayAdapter) IsProviderEnabled(ctx context.Context, provider string) bool {
	return a.fulfillment.IsEnabled(ctx, provider)
}

func (a *checkoutFulfillmentGatewayAdapter) CreateShipment(ctx context.Context, input checkoutapplication.CreateShipmentInput) (checkoutapplication.ShipmentResult, error) {
	result, err := a.fulfillment.CreateShipmentForOrder(ctx, fulfillmentapplication.CreateShipmentInput{
		Provider:       input.Provider,
		DeliveryMethod: input.DeliveryMethod,
		ContactName:    input.ContactName,
		Phone:          input.Phone,
		Email:          input.Email,
		City:           input.Address.City,
		PostalCode:     input.Address.PostalCode,
		Line1:          input.Address.Line1,
		Line2:          input.Address.Line2,
		CountryCode:    input.Address.CountryCode,
		OfficeID:       input.OfficeID,
		RequireCOD:     input.RequireCOD,
		CODAmount:      input.CODAmount,
		Ref1:           input.Ref1,
	})
	if err != nil {
		return checkoutapplication.ShipmentResult{}, err
	}
	return checkoutapplication.ShipmentResult{ShipmentID: result.ShipmentID, ParcelID: result.ParcelID}, nil
}

// fulfillmentOrderGatewayAdapter implements fulfillment's OrderGateway port
// on top of the orders module's application service.
type fulfillmentOrderGatewayAdapter struct {
	orders *ordersapplication.Service
}

func (a *fulfillmentOrderGatewayAdapter) ListAwaitingTracking(ctx context.Context) ([]fulfillmentapplication.TrackedOrderRef, error) {
	orders, err := a.orders.ListAwaitingTracking(ctx)
	if err != nil {
		return nil, err
	}
	refs := make([]fulfillmentapplication.TrackedOrderRef, 0, len(orders))
	for _, o := range orders {
		if o.TrackingNumber == nil {
			continue
		}
		refs = append(refs, fulfillmentapplication.TrackedOrderRef{OrderID: o.ID, ParcelID: *o.TrackingNumber})
	}
	return refs, nil
}

func (a *fulfillmentOrderGatewayAdapter) SetShipmentInfo(ctx context.Context, orderID uuid.UUID, update fulfillmentapplication.ShipmentInfoUpdate) error {
	_, err := a.orders.UpdateFulfillment(ctx, orderID, ordersapplication.UpdateFulfillmentInput{
		Status:         update.OrderStatus,
		Carrier:        update.Carrier,
		TrackingNumber: update.TrackingNumber,
		ShipmentStatus: update.ShipmentStatus,
		ShipmentID:     update.ShipmentID,
	})
	return err
}

// storefrontPromotionsGatewayAdapter implements cataloghttp's
// StorefrontPromotionsGateway on top of the promotions service. It receives
// a map of productID→basePrice so it can compute the discounted effective
// price without importing catalog's domain types.
type storefrontPromotionsGatewayAdapter struct {
	promotions *promotionsapplication.Service
}

func (a *storefrontPromotionsGatewayAdapter) GetEffectivePrices(ctx context.Context, productBasePrices map[uuid.UUID]money.Money) (map[uuid.UUID]cataloghttp.EffectivePromoPrice, error) {
	if len(productBasePrices) == 0 {
		return nil, nil
	}
	productIDs := make([]uuid.UUID, 0, len(productBasePrices))
	for id := range productBasePrices {
		productIDs = append(productIDs, id)
	}
	promos, err := a.promotions.GetEffectivePrices(ctx, productIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID]cataloghttp.EffectivePromoPrice, len(promos))
	for productID, promo := range promos {
		base := productBasePrices[productID]
		discounted := promo.ComputeDiscountedPrice(base)
		if discounted == nil {
			continue
		}
		result[productID] = cataloghttp.EffectivePromoPrice{Price: *discounted, Label: promo.Name}
	}
	return result, nil
}

// navCategoryPromotionsGatewayAdapter implements cataloghttp's
// NavCategoryPromotionsGateway on top of the promotions service.
type navCategoryPromotionsGatewayAdapter struct {
	promotions *promotionsapplication.Service
}

func (a *navCategoryPromotionsGatewayAdapter) GetCategoriesWithActivePromotions(ctx context.Context, categoryIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	return a.promotions.GetCategoriesWithActivePromotions(ctx, categoryIDs)
}

// wishlistPromotionsGatewayAdapter implements wishlisthttp's PromotionsGateway
// using the same promotions service as the storefront gateway.
type wishlistPromotionsGatewayAdapter struct {
	promotions *promotionsapplication.Service
}

func (a *wishlistPromotionsGatewayAdapter) GetEffectivePrices(ctx context.Context, productBasePrices map[uuid.UUID]money.Money) (map[uuid.UUID]wishlisthttp.EffectivePromoPrice, error) {
	if len(productBasePrices) == 0 {
		return nil, nil
	}
	productIDs := make([]uuid.UUID, 0, len(productBasePrices))
	for id := range productBasePrices {
		productIDs = append(productIDs, id)
	}
	promos, err := a.promotions.GetEffectivePrices(ctx, productIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID]wishlisthttp.EffectivePromoPrice, len(promos))
	for productID, promo := range promos {
		base := productBasePrices[productID]
		discounted := promo.ComputeDiscountedPrice(base)
		if discounted == nil {
			continue
		}
		result[productID] = wishlisthttp.EffectivePromoPrice{Price: *discounted, Label: promo.Name}
	}
	return result, nil
}

// checkoutDiscountGatewayAdapter implements checkout's DiscountGateway port
// on top of the promotions module's discount-code service.
type checkoutDiscountGatewayAdapter struct {
	promotions *promotionsapplication.Service
}

func (a *checkoutDiscountGatewayAdapter) ValidateCode(ctx context.Context, code string) (checkoutapplication.DiscountInfo, error) {
	dc, err := a.promotions.ValidateCode(ctx, code)
	if err != nil {
		return checkoutapplication.DiscountInfo{}, err
	}
	return checkoutapplication.DiscountInfo{CodeID: dc.ID, ValuePercent: dc.ValuePercent}, nil
}

func (a *checkoutDiscountGatewayAdapter) UseCode(ctx context.Context, codeID uuid.UUID) error {
	return a.promotions.UseCode(ctx, codeID)
}

// invoicingOrderReaderAdapter satisfies invoicingapplication.OrderReader using
// the orders module's service — avoids a direct cross-module package import.
type invoicingOrderReaderAdapter struct {
	orders *ordersapplication.Service
}

func (a *invoicingOrderReaderAdapter) FindByID(ctx context.Context, id uuid.UUID) (*ordersdomain.Order, error) {
	return a.orders.FindByID(ctx, id)
}

// invoicingProductReaderAdapter satisfies invoicingapplication.ProductInvoiceReader.
type invoicingProductReaderAdapter struct {
	catalog *catalogapplication.ProductService
}

func (a *invoicingProductReaderAdapter) GetTaxGroupID(ctx context.Context, productID uuid.UUID) (*uuid.UUID, error) {
	return a.catalog.GetTaxGroupID(ctx, productID)
}

// invoiceGatewayAdapter satisfies both checkoutapplication.InvoiceGateway and
// ordersapplication.InvoiceGateway using the invoicing module's service.
type invoiceGatewayAdapter struct {
	invoicing *invoicingapplication.Service
}

func (a *invoiceGatewayAdapter) GenerateForOrder(ctx context.Context, orderID uuid.UUID) error {
	_, err := a.invoicing.GenerateForOrder(ctx, orderID, "system")
	return err
}

// deferredInvoiceGateway breaks the circular dependency between ordersService
// (which needs InvoiceGateway) and invoicingService (which needs ordersService
// as its OrderReader). The inner gateway is set after both are constructed.
type deferredInvoiceGateway struct {
	inner *invoiceGatewayAdapter
}

func (d *deferredInvoiceGateway) GenerateForOrder(ctx context.Context, orderID uuid.UUID) error {
	if d.inner == nil {
		return nil
	}
	return d.inner.GenerateForOrder(ctx, orderID)
}

// usersOrderCounterAdapter implements users' OrderCounter port on top of the
// orders module's application service, so the admin User Management page can
// show per-user order counts without users importing orders directly.
type usersOrderCounterAdapter struct {
	orders *ordersapplication.Service
}

func (a *usersOrderCounterAdapter) CountOrdersByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	return a.orders.CountOrdersByUser(ctx, userID)
}

// buildRegistrars wires up domain modules and returns their HTTP route
// registrars, plus the fulfillment service so main.go can launch its
// background tracking poller. Modules are added here as they are
// implemented.
func buildRegistrars(a *app.App) ([]app.RouteRegistrar, *fulfillmentapplication.Service, *checkoutapplication.Service) {
	usersRepo := usersinfra.NewPostgresRepository(a.DB)

	identityRepo := authinfra.NewPostgresIdentityRepository(a.DB)
	sessionRepo := authinfra.NewPostgresSessionRepository(a.DB)
	verifier := googleauth.NewVerifier(a.Config.Google.ClientID)

	ordersRepo := ordersinfra.NewPostgresRepository(a.DB)
	deferredInvoices := &deferredInvoiceGateway{}
	ordersService := ordersapplication.NewService(ordersRepo, deferredInvoices, a.Logger)

	usersService := usersapplication.NewService(usersRepo, &usersOrderCounterAdapter{orders: ordersService})
	provisioner := &userProvisionerAdapter{users: usersService}

	authService := authapplication.NewService(verifier, identityRepo, sessionRepo, provisioner, a.Config.Auth.SessionTTL)

	requireAdmin := func(next http.Handler) http.Handler {
		return authhttp.RequireAuth(authService)(authhttp.RequireAdminAccess()(next))
	}

	authHandler := authhttp.NewHandler(authService, a.Logger)
	usersHandler := usershttp.NewHandler(usersService)
	usersModule := usershttp.NewModule(usersHandler, authhttp.RequireAuth(authService), requireAdmin)

	languageRepo := i18ninfra.NewPostgresLanguageRepository(a.DB)
	languageService := i18napplication.NewLanguageService(languageRepo)
	translationRepo := i18ninfra.NewPostgresTranslationRepository(a.DB)
	translationService := i18napplication.NewTranslationService(translationRepo)
	uiStringRepo := i18ninfra.NewPostgresUIStringRepository(a.DB)
	uiStringService := i18napplication.NewUIStringService(uiStringRepo)
	if err := uiStringService.SeedDefaults(context.Background(), defaultUIStrings); err != nil {
		a.Logger.Error("failed to seed default ui strings", "error", err)
	}

	catalogRepo := cataloginfra.NewPostgresCatalogRepository(a.DB)
	catalogService := catalogapplication.NewCatalogService(catalogRepo)
	catalogHandler := cataloghttp.NewCatalogHandler(catalogService)

	storageClient := storage.NewClient(a.Config.Storage.Endpoint, a.Config.Storage.InsecureSkipTLS, a.Config.Storage.ProjectID)

	categoryRepo := cataloginfra.NewPostgresCategoryRepository(a.DB)
	categoryService := catalogapplication.NewCategoryService(categoryRepo, storageClient, a.Config.Storage.Bucket)
	categoryHandler := cataloghttp.NewCategoryHandler(categoryService)

	productTypeRepo := cataloginfra.NewPostgresProductTypeRepository(a.DB)
	productTypeService := catalogapplication.NewProductTypeService(productTypeRepo)
	productTypeHandler := cataloghttp.NewProductTypeHandler(productTypeService)

	attributeRepo := cataloginfra.NewPostgresAttributeRepository(a.DB)
	attributeService := catalogapplication.NewAttributeService(attributeRepo)
	attributeHandler := cataloghttp.NewAttributeHandler(attributeService)

	productRepo := cataloginfra.NewPostgresProductRepository(a.DB)
	productService := catalogapplication.NewProductService(productRepo, storageClient, a.Config.Storage.Bucket)
	productHandler := cataloghttp.NewProductHandler(productService)

	catalogModule := cataloghttp.NewModule(catalogHandler, categoryHandler, productTypeHandler, attributeHandler, productHandler, requireAdmin)

	invoicingRepo := invoicinginfra.NewPostgresRepository(a.DB)
	invoicingOrderReader := &invoicingOrderReaderAdapter{orders: ordersService}
	invoicingProductReader := &invoicingProductReaderAdapter{catalog: productService}
	invoicingService := invoicingapplication.NewService(invoicingRepo, invoicingOrderReader, invoicingProductReader)
	deferredInvoices.inner = &invoiceGatewayAdapter{invoicing: invoicingService}
	invoicingModule := invoicinghttp.NewModule(invoicinghttp.NewHandler(invoicingService), requireAdmin)

	inventoryRepo := inventoryinfra.NewPostgresRepository(a.DB)
	inventoryService := inventoryapplication.NewService(inventoryRepo)
	inventoryHandler := inventoryhttp.NewHandler(inventoryService)
	inventoryModule := inventoryhttp.NewModule(inventoryHandler, requireAdmin)

	ordersHandler := ordershttp.NewHandler(ordersService)
	ordersModule := ordershttp.NewModule(ordersHandler, authhttp.RequireAuth(authService), requireAdmin)

	paymentsRepo := paymentsinfra.NewPostgresRepository(a.DB)
	paymentsService := paymentsapplication.NewService(paymentsRepo)
	paymentsHandler := paymentshttp.NewHandler(paymentsService)
	paymentsModule := paymentshttp.NewModule(paymentsHandler, authhttp.RequireAuth(authService))

	cartRepo := cartinfra.NewPostgresRepository(a.DB)
	cartService := cartapplication.NewService(cartRepo)
	cartHandler := carthttp.NewHandler(cartService)
	cartModule := carthttp.NewModule(cartHandler, authhttp.OptionalAuth(authService), authhttp.RequireAuth(authService))

	storeSettingsRepo := admininfra.NewPostgresStoreSettingsRepository(a.DB)
	storeSettingsService := adminapplication.NewStoreSettingsService(storeSettingsRepo, storageClient, a.Config.Storage.Bucket).
		WithHeroRepo(storeSettingsRepo).
		WithHomeSectionsRepo(storeSettingsRepo)
	storeSettingsHandler := adminhttp.NewStoreSettingsHandler(storeSettingsService)

	storeAddressRepo := admininfra.NewPostgresStoreAddressRepository(a.DB)
	storeAddressService := adminapplication.NewStoreAddressService(storeAddressRepo, storeSettingsRepo)
	storeAddressHandler := adminhttp.NewStoreAddressHandler(storeAddressService)

	storeDocumentRepo := admininfra.NewPostgresStoreDocumentRepository(a.DB)
	storeDocumentService := adminapplication.NewStoreDocumentService(storeDocumentRepo, storageClient, a.Config.Storage.Bucket)
	storeDocumentHandler := adminhttp.NewStoreDocumentHandler(storeDocumentService)

	adminModule := adminhttp.NewModule(storeSettingsHandler, storeAddressHandler, storeDocumentHandler, requireAdmin)
	adminStorefrontHandler := adminhttp.NewStorefrontHandler(storeSettingsService, storeAddressHandler, storeDocumentHandler)

	languageHandler := i18nhttp.NewLanguageHandler(languageService)
	translationHandler := i18nhttp.NewTranslationHandler(translationService)
	uiStringHandler := i18nhttp.NewUIStringHandler(uiStringService)
	i18nModule := i18nhttp.NewModule(languageHandler, translationHandler, uiStringHandler, requireAdmin)
	i18nStorefrontModule := i18nhttp.NewStorefrontModule(languageHandler, uiStringHandler)

	fulfillmentSettingsRepo := fulfillmentinfra.NewPostgresSettingsRepository(a.DB)
	// Real HTTP client by default; SPEEDY_MODE=fake swaps in a local stand-in
	// that returns canned responses, so delivery methods and tracking can be
	// tested in dev without hitting Speedy or shipping real parcels.
	var fulfillmentSpeedyClient fulfillmentapplication.SpeedyClient = fulfillmentinfra.NewSpeedyHTTPClient()
	if a.Config.Fulfillment.SpeedyMode == app.SpeedyModeFake {
		a.Logger.Warn("using FAKE Speedy client — no real shipments will be created", "speedy_mode", a.Config.Fulfillment.SpeedyMode)
		fulfillmentSpeedyClient = fulfillmentinfra.NewFakeSpeedyClient()
	}
	fulfillmentOrderGateway := &fulfillmentOrderGatewayAdapter{orders: ordersService}
	fulfillmentService := fulfillmentapplication.NewService(fulfillmentSettingsRepo, fulfillmentSpeedyClient, fulfillmentOrderGateway, a.Logger)
	fulfillmentHandler := fulfillmenthttp.NewHandler(fulfillmentService, a.Config.Fulfillment.SpeedyMode == app.SpeedyModeFake)
	fulfillmentModule := fulfillmenthttp.NewModule(fulfillmentHandler, requireAdmin)

	promotionsRepo := promotionsinfra.NewPostgresRepository(a.DB)
	promotionsService := promotionsapplication.NewService(promotionsRepo)
	promotionsHandler := promotionshttp.NewHandler(promotionsService)
	promotionsModule := promotionshttp.NewModule(promotionsHandler, requireAdmin)

	storefrontPromotionsGateway := &storefrontPromotionsGatewayAdapter{promotions: promotionsService}
	navCategoryPromotionsGateway := &navCategoryPromotionsGatewayAdapter{promotions: promotionsService}
	storefrontHandler := cataloghttp.NewStorefrontHandler(productTypeService, categoryService, productService, catalogService, translationService, storefrontPromotionsGateway, navCategoryPromotionsGateway)

	checkoutCartGateway := &checkoutCartGatewayAdapter{cart: cartService}
	checkoutInventoryGateway := &checkoutInventoryGatewayAdapter{inventory: inventoryService}
	checkoutUserGateway := &checkoutUserGatewayAdapter{users: usersService}
	checkoutOrderGateway := &checkoutOrderGatewayAdapter{orders: ordersService}
	// Real Revolut client when an API key is configured; otherwise the mock
	// gateway keeps local/devbox checkout working without a merchant account.
	var checkoutPaymentGateway checkoutapplication.PaymentGateway = checkoutinfra.NewMockRevolutGateway()
	if a.Config.Payments.RevolutAPIKey != "" {
		checkoutPaymentGateway = checkoutinfra.NewRevolutGateway(
			a.Config.Payments.RevolutBaseURL(),
			a.Config.Payments.RevolutAPIKey,
			a.Config.Payments.RevolutAPIVersion,
			a.Logger,
		)
	} else {
		a.Logger.Warn("REVOLUT_API_KEY not set — using mock Revolut gateway; no real card payments will be taken")
	}
	checkoutFulfillmentGateway := &checkoutFulfillmentGatewayAdapter{fulfillment: fulfillmentService}
	checkoutDiscountGateway := &checkoutDiscountGatewayAdapter{promotions: promotionsService}
	checkoutWebhookStore := checkoutinfra.NewPostgresWebhookEventStore(a.DB)
	checkoutService := checkoutapplication.NewService(checkoutCartGateway, checkoutInventoryGateway, checkoutUserGateway, checkoutOrderGateway, checkoutPaymentGateway, checkoutFulfillmentGateway, checkoutDiscountGateway, deferredInvoices, checkoutWebhookStore, a.Logger)
	checkoutHandler := checkouthttp.NewHandler(checkoutService, a.Config.Payments.RevolutWebhookSecret)
	checkoutModule := checkouthttp.NewModule(checkoutHandler, authhttp.OptionalAuth(authService), requireAdmin)

	wishlistRepo := wishlistinfra.NewPostgresRepository(a.DB)
	wishlistService := wishlistapplication.NewService(wishlistRepo)
	wishlistPromotionsGateway := &wishlistPromotionsGatewayAdapter{promotions: promotionsService}
	wishlistHandler := wishlisthttp.NewHandler(wishlistService, wishlistPromotionsGateway)
	wishlistModule := wishlisthttp.NewModule(wishlistHandler, authhttp.RequireAuth(authService))

	return []app.RouteRegistrar{
		authHandler,
		usersModule,
		catalogModule,
		storefrontHandler,
		inventoryModule,
		ordersModule,
		paymentsModule,
		cartModule,
		adminModule,
		adminStorefrontHandler,
		checkoutModule,
		fulfillmentModule,
		promotionsModule,
		i18nModule,
		i18nStorefrontModule,
		wishlistModule,
		invoicingModule,
	}, fulfillmentService, checkoutService
}

// defaultUIStrings seeds the English baseline for the static-text
// translation system. New keys are added here as they're introduced in the
// frontend; SeedDefaults is idempotent (ON CONFLICT DO NOTHING) so this is
// safe to extend and re-run on every startup. Admins translate these into
// other languages via the admin Translations editor.
var defaultUIStrings = map[string]string{
	// Navigation
	"nav.new_arrivals": "New Arrivals",
	"nav.shop":         "Shop",
	"nav.shop_all":     "Shop All",

	// Footer
	"footer.help":                "Help",
	"footer.shipping_returns":    "Shipping & Returns",
	"footer.size_guide":          "Size Guide",
	"footer.contact_us":          "Contact Us",
	"footer.faq":                 "FAQ",
	"footer.company":             "Company",
	"footer.about":               "About",
	"footer.sustainability":      "Sustainability",
	"footer.privacy_policy":      "Privacy Policy",
	"footer.terms_of_service":    "Terms of Service",
	"footer.free_shipping":       "Free shipping over $100",
	"footer.company_description": "Clothing, jewelry, bags, and accessories, thoughtfully made and delivered with care.",
	"footer.all_rights_reserved": "All rights reserved.",

	// Header
	"header.toggle_menu":        "Toggle menu",
	"header.search_placeholder": "Search products…",
	"header.search":             "Search",
	"header.account":            "Account",
	"header.wishlist":           "Wishlist",
	"header.cart":               "Cart",

	// Common UI
	"common.loading":           "Loading…",
	"common.saving":            "Saving…",
	"common.back":              "Back",
	"common.cancel":            "Cancel",
	"common.save":              "Save",
	"common.optional":          "Optional",
	"common.default_badge":     "Default",
	"common.continue_shopping": "Continue Shopping",
	"common.full_name":         "Full name",
	"common.email":             "Email",
	"common.phone":             "Phone",
	"common.address_line1":     "Address line 1",
	"common.address_line2":     "Address line 2",
	"common.city":              "City",
	"common.region":            "Region / State",
	"common.postal_code":       "Postal code",
	"common.country":           "Country",
	"common.select_country":    "Select a country",
	"common.expiry_month":      "Expiry month",
	"common.expiry_year":       "Expiry year",

	// Shop / Browse
	"shop.all_products":       "All Products",
	"shop.search_results_for": "Search results for",
	"shop.filters":            "Filters",
	"shop.clear_filters":      "Clear all",
	"shop.no_products":        "No products match these filters yet.",
	"shop.load_error":         "Could not load products.",
	"shop.filter_type":        "Type",
	"shop.filter_category":    "Category",

	// Home page sections
	"home.new_in":           "New In",
	"home.new_arrivals":     "New Arrivals",
	"home.browse":           "Browse",
	"home.shop_by_category": "Shop by Category",

	// Product
	"product.add_to_cart":       "Add to Cart",
	"product.out_of_stock":      "Out of Stock",
	"product.select_size":       "Select Size",
	"product.adding_to_cart":    "Adding…",
	"product.added_to_cart":     "Added",
	"product.add_to_cart_error": "Could not add this item to your cart.",
	"product.load_error":        "Could not load this product.",
	"product.wishlist_add":      "Add to wishlist",
	"product.sold_out":          "Sold Out",
	"product.available":         "Available",
	"product.sale_badge":        "Sale",

	// Cart
	"cart.title":         "Your Cart",
	"cart.empty":         "Your cart is empty",
	"cart.checkout":      "Checkout",
	"cart.summary":       "Summary",
	"cart.subtotal":      "Subtotal",
	"cart.update_error":  "Could not update quantity.",
	"cart.remove_error":  "Could not remove this item.",
	"cart.decrease_qty":  "Decrease quantity",
	"cart.increase_qty":  "Increase quantity",
	"cart.remove_item":   "Remove item",
	"cart.only":          "Only",
	"cart.left_in_stock": "left in stock",

	// Wishlist
	"wishlist.title":       "Wishlist",
	"wishlist.empty_title": "Your wishlist is empty",
	"wishlist.empty_desc":  "Save items you love by tapping the heart on any product.",
	"wishlist.sizes_label": "Sizes:",
	"wishlist.remove":      "Remove from wishlist",

	// Checkout steps
	"checkout.title":         "Checkout",
	"checkout.place_order":   "Place Order",
	"checkout.step_details":  "Details",
	"checkout.step_delivery": "Delivery",
	"checkout.step_payment":  "Payment",
	"checkout.step_review":   "Review",

	// Checkout — contact & shipping step
	"checkout.contact_shipping":         "Contact & Shipping",
	"checkout.signin_prompt":            "Have an account? Sign in for a faster checkout and to track this order.",
	"checkout.login_register":           "Log In / Register",
	"checkout.shipping_address":         "Shipping address",
	"checkout.enter_new_address":        "Enter a new address",
	"checkout.billing_same_as_shipping": "Billing address same as shipping",
	"checkout.billing_address":          "Billing address",
	"checkout.continue_to_delivery":     "Continue to Delivery",
	"checkout.contact_required_error":   "Full name and email are required.",
	"checkout.shipping_required_error":  "A complete shipping address with a country is required.",
	"checkout.billing_required_error":   "A complete billing address with a country is required.",

	// Checkout — delivery step
	"checkout.delivery_method":        "Delivery Method",
	"checkout.free":                   "Free",
	"checkout.choose_locker":          "Choose a locker",
	"checkout.enter_city_for_lockers": "Enter a shipping city to see nearby lockers.",
	"checkout.loading_lockers":        "Loading lockers…",
	"checkout.no_lockers_found":       "No lockers found for this city.",
	"checkout.select_locker":          "Select a locker",
	"checkout.load_lockers_error":     "Could not load lockers for this city.",
	"checkout.continue_to_payment":    "Continue to Payment",

	// Checkout — payment step
	"checkout.payment_method":        "Payment Method",
	"checkout.cash_on_delivery":      "Cash on Delivery",
	"checkout.cash_on_delivery_desc": "Pay in cash when your courier delivers the order.",
	"checkout.card_on_easybox":       "Card on EasyBox Pickup",
	"checkout.card_on_easybox_desc":  "Pay by card at the locker when you collect your order.",
	"checkout.card_online":           "Pay by Card Online",
	"checkout.card_online_desc":      "Pay securely now with your card.",
	"checkout.card_number":           "Card number",
	"checkout.card_mock_hint":        "Mock payment — any number works, except one ending in 0000.",
	"checkout.cvv":                   "CVV",
	"checkout.review_order":          "Review Order",
	"checkout.processing_payment":    "Processing payment…",
	"checkout.pay":                   "Pay",
	"checkout.payment_failed_error":  "Payment could not be processed. Please check your card details and try again.",
	"checkout.place_order_error":     "Could not place your order. Please try again.",

	// Checkout — review & summary
	"checkout.review_complete": "Review & Complete",
	"checkout.order_summary":   "Order Summary",
	"checkout.delivery_label":  "Delivery",
	"checkout.payment_label":   "Payment",
	"checkout.subtotal":        "Subtotal",
	"checkout.discount_label":  "Discount",
	"checkout.delivery_fee":    "Delivery fee",
	"checkout.total":           "Total",
	"checkout.placing_order":   "Placing order…",
	"checkout.complete_order":  "Complete Order",

	// Checkout — confirmation
	"checkout.order_placed":           "Order Placed",
	"checkout.order_confirmed_prefix": "Thank you! Your order",
	"checkout.order_confirmed_suffix": "has been",
	"checkout.order_paid_placed":      "paid and placed",
	"checkout.order_placed_fallback":  "placed",

	// Checkout — discount code
	"checkout.discount_code":    "Discount code",
	"checkout.apply":            "Apply",
	"checkout.invalid_discount": "This discount code is invalid or has expired.",
	"checkout.discount_error":   "Could not validate the discount code.",

	// Login
	"login.welcome_back":      "Welcome back",
	"login.continue_browsing": "Continue browsing",
	"login.hero_eyebrow":      "New Season",
	"login.hero_heading":      "Crafted pieces, made to last.",
	"login.hero_subtext":      "Sign in to track orders, save favorites, and check out faster.",
	"login.subheading":        "Sign in to access your account, orders, and wishlist.",
	"login.terms_prefix":      "By continuing, you agree to",
	"login.without_signin":    "without signing in",

	// About
	"about.placeholder": "More information about us is coming soon.",

	// Account — orders
	"account.orders.load_error":  "Could not load your orders.",
	"account.orders.empty_title": "No orders yet",
	"account.orders.empty_desc":  "Your placed orders will show up here.",
	"account.orders.col_order":   "Order",
	"account.orders.col_date":    "Date",
	"account.orders.col_status":  "Status",
	"account.orders.col_items":   "Items",
	"account.orders.col_total":   "Total",
	"account.orders.hide_items":  "Hide order items",
	"account.orders.show_items":  "Show order items",
	"account.orders.carrier":     "Carrier",

	// Account — personal info
	"account.profile.load_error":   "Could not load your profile.",
	"account.profile.save_error":   "Could not save changes.",
	"account.profile.saved":        "Saved",
	"account.profile.save_changes": "Save Changes",
	"account.profile.your_details": "Your Details",
	"account.profile.email_hint":   "Managed via your Google sign-in",

	// Account — addresses
	"account.addresses.load_error":     "Could not load addresses.",
	"account.addresses.delete_error":   "Could not delete address.",
	"account.addresses.add_button":     "Add Address",
	"account.addresses.empty_title":    "No addresses yet",
	"account.addresses.empty_desc":     "Add a shipping address to get started.",
	"account.addresses.address_label":  "Address",
	"account.addresses.edit":           "Edit address",
	"account.addresses.delete":         "Delete address",
	"account.addresses.modal_add":      "Add Address",
	"account.addresses.modal_edit":     "Edit Address",
	"account.addresses.label":          "Label",
	"account.addresses.label_hint":     "Optional, e.g. Home or Office",
	"account.addresses.required_error": "Address line 1, city, and postal code are required.",
	"account.addresses.set_default":    "Set as default address",
	"account.addresses.save_error":     "Could not save changes. Try again.",
	"account.addresses.create_error":   "Could not create address. Try again.",

	// Account — payment methods
	"account.payment.load_error":        "Could not load payment methods.",
	"account.payment.remove_error":      "Could not remove payment method.",
	"account.payment.add_button":        "Add Card",
	"account.payment.empty_title":       "No payment methods yet",
	"account.payment.empty_desc":        "Add a card to use at checkout.",
	"account.payment.edit":              "Edit payment method",
	"account.payment.remove":            "Remove payment method",
	"account.payment.modal_add":         "Add Card",
	"account.payment.modal_edit":        "Edit Card",
	"account.payment.security_note":     "For your security, we only store the card brand, last 4 digits, and expiry — never the full card number.",
	"account.payment.brand":             "Card brand",
	"account.payment.brand_placeholder": "Choose a brand…",
	"account.payment.last4":             "Last 4 digits",
	"account.payment.brand_required":    "Card brand is required.",
	"account.payment.last4_error":       "Last 4 digits must be exactly 4 numbers.",
	"account.payment.set_default":       "Set as default payment method",
	"account.payment.save_error":        "Could not save changes. Try again.",
	"account.payment.add_error":         "Could not add card. Try again.",
	"account.payment.expires":           "Expires",
}
