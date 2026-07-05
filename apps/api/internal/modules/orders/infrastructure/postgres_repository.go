package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/orders/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/orders/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const orderColumns = `
	id, user_id, order_number, status, total_amount, total_currency, placed_at,
	contact_name, contact_email, contact_phone,
	shipping_recipient_name, shipping_phone, shipping_line1, shipping_line2, shipping_city, shipping_region, shipping_postal_code, shipping_country_code,
	billing_recipient_name, billing_phone, billing_line1, billing_line2, billing_city, billing_region, billing_postal_code, billing_country_code,
	delivery_method, delivery_fee_amount, delivery_fee_currency, payment_method,
	carrier, tracking_number, shipment_status, speedy_shipment_id, delivery_office_id, viewed_by_admin_at, reservation_id,
	discount_code, discount_amount_minor, discount_amount_currency,
	created_at, updated_at`

func (r *PostgresRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Order, error) {
	rows, err := r.db.Query(ctx, `SELECT `+orderColumns+` FROM orders WHERE user_id = $1 ORDER BY placed_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, *o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range orders {
		if err := r.attachItemsAndPayment(ctx, &orders[i]); err != nil {
			return nil, err
		}
	}
	return orders, nil
}

// AdminList supports the two filters the admin orders page needs: a status
// filter and the "unviewed only" toggle behind the sidebar's unread badge.
func (r *PostgresRepository) CountByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM orders WHERE user_id = $1`, userID).Scan(&count)
	return count, err
}

func (r *PostgresRepository) AdminList(ctx context.Context, filter application.AdminListOrdersFilter) ([]domain.Order, error) {
	query := `SELECT ` + orderColumns + ` FROM orders WHERE ($1::text IS NULL OR status = $1) AND ($2::bool IS FALSE OR viewed_by_admin_at IS NULL) ORDER BY placed_at DESC`

	rows, err := r.db.Query(ctx, query, filter.Status, filter.UnviewedOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, *o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range orders {
		if err := r.attachItemsAndPayment(ctx, &orders[i]); err != nil {
			return nil, err
		}
	}
	return orders, nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	row := r.db.QueryRow(ctx, `SELECT `+orderColumns+` FROM orders WHERE id = $1`, id)
	order, err := scanOrder(row)
	if err != nil {
		return nil, err
	}
	if err := r.attachItemsAndPayment(ctx, order); err != nil {
		return nil, err
	}
	return order, nil
}

func (r *PostgresRepository) attachItemsAndPayment(ctx context.Context, order *domain.Order) error {
	items, err := r.itemsFor(ctx, order.ID)
	if err != nil {
		return err
	}
	order.Items = items

	payment, err := r.paymentFor(ctx, order.ID)
	if err != nil {
		return err
	}
	order.Payment = payment
	return nil
}

func (r *PostgresRepository) Create(ctx context.Context, order domain.Order) (*domain.Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		INSERT INTO orders (
			user_id, order_number, status, total_amount, total_currency, placed_at,
			contact_name, contact_email, contact_phone,
			shipping_recipient_name, shipping_phone, shipping_line1, shipping_line2, shipping_city, shipping_region, shipping_postal_code, shipping_country_code,
			billing_recipient_name, billing_phone, billing_line1, billing_line2, billing_city, billing_region, billing_postal_code, billing_country_code,
			delivery_method, delivery_fee_amount, delivery_fee_currency, payment_method, delivery_office_id, reservation_id,
			discount_code, discount_amount_minor, discount_amount_currency
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34)
		RETURNING `+orderColumns,
		order.UserID, order.OrderNumber, order.Status, order.Total.AmountMinor, order.Total.Currency, order.PlacedAt,
		order.ContactName, order.ContactEmail, order.ContactPhone,
		order.ShippingAddress.RecipientName, order.ShippingAddress.Phone, order.ShippingAddress.Line1, order.ShippingAddress.Line2,
		order.ShippingAddress.City, order.ShippingAddress.Region, order.ShippingAddress.PostalCode, order.ShippingAddress.CountryCode,
		order.BillingAddress.RecipientName, order.BillingAddress.Phone, order.BillingAddress.Line1, order.BillingAddress.Line2,
		order.BillingAddress.City, order.BillingAddress.Region, order.BillingAddress.PostalCode, order.BillingAddress.CountryCode,
		order.DeliveryMethod, order.DeliveryFee.AmountMinor, order.DeliveryFee.Currency, order.PaymentMethod, order.DeliveryOfficeID, order.ReservationID,
		order.DiscountCode, discountAmountMinor(order.DiscountAmount), discountAmountCurrency(order.DiscountAmount))

	created, err := scanOrder(row)
	if err != nil {
		return nil, err
	}

	for _, item := range order.Items {
		itemRow := tx.QueryRow(ctx, `
			INSERT INTO order_items (order_id, product_id, product_name, variant_label, quantity, unit_price_amount, unit_price_currency)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id, order_id, product_id, product_name, COALESCE(variant_label, ''), quantity, unit_price_amount, unit_price_currency, created_at`,
			created.ID, item.ProductID, item.ProductName, item.VariantLabel, item.Quantity, item.UnitPrice.AmountMinor, item.UnitPrice.Currency)

		scannedItem, err := scanOrderItem(itemRow)
		if err != nil {
			return nil, err
		}
		created.Items = append(created.Items, *scannedItem)
	}

	if order.Payment != nil {
		paymentRow := tx.QueryRow(ctx, `
			INSERT INTO order_payments (order_id, provider, provider_reference, status, amount_minor, currency)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, order_id, provider, COALESCE(provider_reference, ''), status, amount_minor, currency, created_at`,
			created.ID, order.Payment.Provider, order.Payment.ProviderReference, order.Payment.Status,
			order.Payment.Amount.AmountMinor, order.Payment.Amount.Currency)

		payment, err := scanOrderPayment(paymentRow)
		if err != nil {
			return nil, err
		}
		created.Payment = payment
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return created, nil
}

func (r *PostgresRepository) UpdateFulfillment(ctx context.Context, id uuid.UUID, input application.UpdateFulfillmentInput) (*domain.Order, error) {
	current, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	status := string(current.Status)
	if input.Status != nil {
		status = *input.Status
	}
	carrier := current.Carrier
	if input.Carrier != nil {
		carrier = input.Carrier
	}
	trackingNumber := current.TrackingNumber
	if input.TrackingNumber != nil {
		trackingNumber = input.TrackingNumber
	}
	shipmentStatus := current.ShipmentStatus
	if input.ShipmentStatus != nil {
		shipmentStatus = input.ShipmentStatus
	}
	shipmentID := current.SpeedyShipmentID
	if input.ShipmentID != nil {
		shipmentID = input.ShipmentID
	}

	row := r.db.QueryRow(ctx, `
		UPDATE orders SET status = $2, carrier = $3, tracking_number = $4, shipment_status = $5, speedy_shipment_id = $6, updated_at = NOW()
		WHERE id = $1
		RETURNING `+orderColumns,
		id, status, carrier, trackingNumber, shipmentStatus, shipmentID)

	updated, err := scanOrder(row)
	if err != nil {
		return nil, err
	}
	if err := r.attachItemsAndPayment(ctx, updated); err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *PostgresRepository) ListAwaitingTracking(ctx context.Context) ([]domain.Order, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+orderColumns+` FROM orders
		WHERE carrier IS NOT NULL AND tracking_number IS NOT NULL
		AND status NOT IN ('delivered', 'cancelled')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, *o)
	}
	return orders, rows.Err()
}

func (r *PostgresRepository) MarkViewed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE orders SET viewed_by_admin_at = NOW() WHERE id = $1 AND viewed_by_admin_at IS NULL`, id)
	return err
}

func (r *PostgresRepository) CountUnviewed(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM orders WHERE viewed_by_admin_at IS NULL`).Scan(&count)
	return count, err
}

func (r *PostgresRepository) itemsFor(ctx context.Context, orderID uuid.UUID) ([]domain.OrderItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, order_id, product_id, product_name, COALESCE(variant_label, ''), quantity, unit_price_amount, unit_price_currency, created_at
		FROM order_items WHERE order_id = $1 ORDER BY created_at`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.OrderItem{}
	for rows.Next() {
		item, err := scanOrderItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) paymentFor(ctx context.Context, orderID uuid.UUID) (*domain.OrderPayment, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, order_id, provider, COALESCE(provider_reference, ''), status, amount_minor, currency, created_at
		FROM order_payments WHERE order_id = $1 ORDER BY created_at DESC LIMIT 1`, orderID)
	payment, err := scanOrderPayment(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return payment, nil
}

func (r *PostgresRepository) Stats(ctx context.Context, since time.Time) (application.OrderStats, error) {
	var stats application.OrderStats

	var revenueAmount int64
	var revenueCurrency *string
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(total_amount), 0), MAX(total_currency)
		FROM orders WHERE placed_at >= $1`, since,
	).Scan(&stats.OrderCount, &revenueAmount, &revenueCurrency); err != nil {
		return stats, err
	}
	currency := "EUR"
	if revenueCurrency != nil {
		currency = *revenueCurrency
	}
	stats.Revenue = money.Money{AmountMinor: revenueAmount, Currency: currency}
	if stats.OrderCount > 0 {
		stats.AvgOrderValue = money.Money{AmountMinor: revenueAmount / int64(stats.OrderCount), Currency: currency}
	} else {
		stats.AvgOrderValue = money.Money{AmountMinor: 0, Currency: currency}
	}

	statusRows, err := r.db.Query(ctx, `
		SELECT status, COUNT(*) FROM orders WHERE placed_at >= $1 GROUP BY status ORDER BY COUNT(*) DESC`, since)
	if err != nil {
		return stats, err
	}
	stats.StatusBreakdown, err = scanCountBreakdown(statusRows)
	if err != nil {
		return stats, err
	}

	cityRows, err := r.db.Query(ctx, `
		SELECT shipping_city, COUNT(*) FROM orders WHERE placed_at >= $1 AND shipping_city <> ''
		GROUP BY shipping_city ORDER BY COUNT(*) DESC LIMIT 15`, since)
	if err != nil {
		return stats, err
	}
	stats.ByCity, err = scanCountBreakdown(cityRows)
	if err != nil {
		return stats, err
	}

	countryRows, err := r.db.Query(ctx, `
		SELECT shipping_country_code, COUNT(*) FROM orders WHERE placed_at >= $1 AND shipping_country_code <> ''
		GROUP BY shipping_country_code ORDER BY COUNT(*) DESC`, since)
	if err != nil {
		return stats, err
	}
	stats.ByCountry, err = scanCountBreakdown(countryRows)
	if err != nil {
		return stats, err
	}

	deliveryRows, err := r.db.Query(ctx, `
		SELECT delivery_method, COUNT(*) FROM orders WHERE placed_at >= $1
		GROUP BY delivery_method ORDER BY COUNT(*) DESC`, since)
	if err != nil {
		return stats, err
	}
	stats.ByDeliveryMethod, err = scanCountBreakdown(deliveryRows)
	if err != nil {
		return stats, err
	}

	dailyRows, err := r.db.Query(ctx, `
		SELECT date_trunc('day', placed_at) AS day, COUNT(*), COALESCE(SUM(total_amount), 0)
		FROM orders WHERE placed_at >= $1 GROUP BY day ORDER BY day`, since)
	if err != nil {
		return stats, err
	}
	defer dailyRows.Close()
	for dailyRows.Next() {
		var d application.DailyOrderCount
		var amount int64
		if err := dailyRows.Scan(&d.Date, &d.Count, &amount); err != nil {
			return stats, err
		}
		d.Revenue = money.Money{AmountMinor: amount, Currency: currency}
		stats.DailyCounts = append(stats.DailyCounts, d)
	}
	if err := dailyRows.Err(); err != nil {
		return stats, err
	}

	return stats, nil
}

func scanCountBreakdown(rows pgx.Rows) ([]application.CountBreakdown, error) {
	defer rows.Close()
	result := []application.CountBreakdown{}
	for rows.Next() {
		var b application.CountBreakdown
		if err := rows.Scan(&b.Label, &b.Count); err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, rows.Err()
}

func scanOrder(row pgx.Row) (*domain.Order, error) {
	var o domain.Order
	var status string
	var amount int64
	var currency string
	var contactName, contactEmail, contactPhone string
	var shipRecipient, shipPhone, shipLine1, shipLine2, shipCity, shipRegion, shipPostal, shipCountry string
	var billRecipient, billPhone, billLine1, billLine2, billCity, billRegion, billPostal, billCountry string
	var deliveryMethod, paymentMethod string
	var deliveryFeeAmount int64
	var deliveryFeeCurrency string
	var discountAmountMinorCol *int64
	var discountAmountCurrencyCol *string

	err := row.Scan(
		&o.ID, &o.UserID, &o.OrderNumber, &status, &amount, &currency, &o.PlacedAt,
		&contactName, &contactEmail, &contactPhone,
		&shipRecipient, &shipPhone, &shipLine1, &shipLine2, &shipCity, &shipRegion, &shipPostal, &shipCountry,
		&billRecipient, &billPhone, &billLine1, &billLine2, &billCity, &billRegion, &billPostal, &billCountry,
		&deliveryMethod, &deliveryFeeAmount, &deliveryFeeCurrency, &paymentMethod,
		&o.Carrier, &o.TrackingNumber, &o.ShipmentStatus, &o.SpeedyShipmentID, &o.DeliveryOfficeID, &o.ViewedByAdminAt, &o.ReservationID,
		&o.DiscountCode, &discountAmountMinorCol, &discountAmountCurrencyCol,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}

	o.Status = domain.OrderStatus(status)
	o.Total = money.Money{AmountMinor: amount, Currency: currency}
	o.ContactName, o.ContactEmail, o.ContactPhone = contactName, contactEmail, contactPhone
	o.ShippingAddress = domain.OrderAddress{
		RecipientName: shipRecipient, Phone: shipPhone, Line1: shipLine1, Line2: shipLine2,
		City: shipCity, Region: shipRegion, PostalCode: shipPostal, CountryCode: shipCountry,
	}
	o.BillingAddress = domain.OrderAddress{
		RecipientName: billRecipient, Phone: billPhone, Line1: billLine1, Line2: billLine2,
		City: billCity, Region: billRegion, PostalCode: billPostal, CountryCode: billCountry,
	}
	o.DeliveryMethod = deliveryMethod
	o.DeliveryFee = money.Money{AmountMinor: deliveryFeeAmount, Currency: deliveryFeeCurrency}
	o.PaymentMethod = paymentMethod
	if discountAmountMinorCol != nil && discountAmountCurrencyCol != nil {
		m := money.Money{AmountMinor: *discountAmountMinorCol, Currency: *discountAmountCurrencyCol}
		o.DiscountAmount = &m
	}
	return &o, nil
}

func discountAmountMinor(m *money.Money) *int64 {
	if m == nil {
		return nil
	}
	return &m.AmountMinor
}

func discountAmountCurrency(m *money.Money) *string {
	if m == nil {
		return nil
	}
	return &m.Currency
}

func scanOrderItem(row pgx.Row) (*domain.OrderItem, error) {
	var item domain.OrderItem
	var amount int64
	var currency string
	if err := row.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.ProductName, &item.VariantLabel, &item.Quantity,
		&amount, &currency, &item.CreatedAt); err != nil {
		return nil, err
	}
	item.UnitPrice = money.Money{AmountMinor: amount, Currency: currency}
	return &item, nil
}

func scanOrderPayment(row pgx.Row) (*domain.OrderPayment, error) {
	var p domain.OrderPayment
	var amount int64
	var currency string
	if err := row.Scan(&p.ID, &p.OrderID, &p.Provider, &p.ProviderReference, &p.Status, &amount, &currency, &p.CreatedAt); err != nil {
		return nil, err
	}
	p.Amount = money.Money{AmountMinor: amount, Currency: currency}
	return &p, nil
}
