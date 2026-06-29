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
	ordersapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/orders/application"
	ordersdomain "github.com/adotomov/fashion-store/apps/api/internal/modules/orders/domain"
	ordersinfra "github.com/adotomov/fashion-store/apps/api/internal/modules/orders/infrastructure"
	ordershttp "github.com/adotomov/fashion-store/apps/api/internal/modules/orders/transport/http"
	paymentsapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/payments/application"
	paymentsinfra "github.com/adotomov/fashion-store/apps/api/internal/modules/payments/infrastructure"
	paymentshttp "github.com/adotomov/fashion-store/apps/api/internal/modules/payments/transport/http"
	usersapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/users/application"
	usersinfra "github.com/adotomov/fashion-store/apps/api/internal/modules/users/infrastructure"
	usershttp "github.com/adotomov/fashion-store/apps/api/internal/modules/users/transport/http"
	"github.com/adotomov/fashion-store/apps/api/internal/platform/googleauth"
	"github.com/adotomov/fashion-store/apps/api/internal/platform/storage"
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
func buildRegistrars(a *app.App) ([]app.RouteRegistrar, *fulfillmentapplication.Service) {
	usersRepo := usersinfra.NewPostgresRepository(a.DB)

	identityRepo := authinfra.NewPostgresIdentityRepository(a.DB)
	sessionRepo := authinfra.NewPostgresSessionRepository(a.DB)
	verifier := googleauth.NewVerifier(a.Config.Google.ClientID)

	ordersRepo := ordersinfra.NewPostgresRepository(a.DB)
	ordersService := ordersapplication.NewService(ordersRepo)

	usersService := usersapplication.NewService(usersRepo, &usersOrderCounterAdapter{orders: ordersService})
	provisioner := &userProvisionerAdapter{users: usersService}

	authService := authapplication.NewService(verifier, identityRepo, sessionRepo, provisioner, a.Config.Auth.SessionTTL)

	requireAdmin := func(next http.Handler) http.Handler {
		return authhttp.RequireAuth(authService)(authhttp.RequireRole("admin")(next))
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

	storageClient := storage.NewClient(a.Config.Storage.Endpoint, a.Config.Storage.InsecureSkipTLS)

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
	storefrontHandler := cataloghttp.NewStorefrontHandler(productTypeService, categoryService, productService, catalogService, translationService)

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
	storeSettingsService := adminapplication.NewStoreSettingsService(storeSettingsRepo, storageClient, a.Config.Storage.Bucket)
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
	fulfillmentSpeedyClient := fulfillmentinfra.NewSpeedyHTTPClient()
	fulfillmentOrderGateway := &fulfillmentOrderGatewayAdapter{orders: ordersService}
	fulfillmentService := fulfillmentapplication.NewService(fulfillmentSettingsRepo, fulfillmentSpeedyClient, fulfillmentOrderGateway, a.Logger)
	fulfillmentHandler := fulfillmenthttp.NewHandler(fulfillmentService)
	fulfillmentModule := fulfillmenthttp.NewModule(fulfillmentHandler, requireAdmin)

	checkoutCartGateway := &checkoutCartGatewayAdapter{cart: cartService}
	checkoutInventoryGateway := &checkoutInventoryGatewayAdapter{inventory: inventoryService}
	checkoutUserGateway := &checkoutUserGatewayAdapter{users: usersService}
	checkoutOrderGateway := &checkoutOrderGatewayAdapter{orders: ordersService}
	checkoutPaymentGateway := checkoutinfra.NewMockRevolutGateway()
	checkoutFulfillmentGateway := &checkoutFulfillmentGatewayAdapter{fulfillment: fulfillmentService}
	checkoutService := checkoutapplication.NewService(checkoutCartGateway, checkoutInventoryGateway, checkoutUserGateway, checkoutOrderGateway, checkoutPaymentGateway, checkoutFulfillmentGateway, a.Logger)
	checkoutHandler := checkouthttp.NewHandler(checkoutService)
	checkoutModule := checkouthttp.NewModule(checkoutHandler, authhttp.OptionalAuth(authService))

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
		i18nModule,
		i18nStorefrontModule,
	}, fulfillmentService
}

// defaultUIStrings seeds the English baseline for the static-text
// translation system. New keys are added here as they're introduced in the
// frontend; SeedDefaults is idempotent (ON CONFLICT DO NOTHING) so this is
// safe to extend and re-run on every startup. Admins translate these into
// other languages via the admin Translations editor.
var defaultUIStrings = map[string]string{
	"nav.new_arrivals":        "New Arrivals",
	"nav.shop":                "Shop",
	"footer.help":             "Help",
	"footer.shipping_returns": "Shipping & Returns",
	"footer.size_guide":       "Size Guide",
	"footer.contact_us":       "Contact Us",
	"footer.faq":              "FAQ",
	"footer.company":          "Company",
	"footer.about":            "About",
	"footer.sustainability":   "Sustainability",
	"footer.privacy_policy":   "Privacy Policy",
	"footer.terms_of_service": "Terms of Service",
	"footer.free_shipping":    "Free shipping over $100",
	"product.add_to_cart":     "Add to Cart",
	"product.out_of_stock":    "Out of Stock",
	"product.select_size":     "Select Size",
	"cart.title":              "Your Cart",
	"cart.empty":              "Your cart is empty",
	"cart.checkout":           "Checkout",
	"checkout.title":          "Checkout",
	"checkout.place_order":    "Place Order",
	"login.welcome_back":      "Welcome back",
	"login.continue_browsing": "Continue browsing",
}
