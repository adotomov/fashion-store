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
	carrier, tracking_number, shipment_status, speedy_shipment_id, delivery_office_id, viewed_by_admin_at, reservation_id, cart_guest_token,
	discount_code, discount_amount_minor, discount_amount_currency,
	created_at, updated_at`

const paymentColumns = `
	id, order_id, provider, COALESCE(provider_order_id, ''), COALESCE(provider_reference, ''),
	status, amount_minor, currency, captured_minor, refunded_minor, created_at`

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

func (r *PostgresRepository) FindByOrderNumber(ctx context.Context, orderNumber string) (*domain.Order, error) {
	row := r.db.QueryRow(ctx, `SELECT `+orderColumns+` FROM orders WHERE order_number = $1`, orderNumber)
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
			delivery_method, delivery_fee_amount, delivery_fee_currency, payment_method, delivery_office_id, reservation_id, cart_guest_token,
			discount_code, discount_amount_minor, discount_amount_currency
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35)
		RETURNING `+orderColumns,
		order.UserID, order.OrderNumber, order.Status, order.Total.AmountMinor, order.Total.Currency, order.PlacedAt,
		order.ContactName, order.ContactEmail, order.ContactPhone,
		order.ShippingAddress.RecipientName, order.ShippingAddress.Phone, order.ShippingAddress.Line1, order.ShippingAddress.Line2,
		order.ShippingAddress.City, order.ShippingAddress.Region, order.ShippingAddress.PostalCode, order.ShippingAddress.CountryCode,
		order.BillingAddress.RecipientName, order.BillingAddress.Phone, order.BillingAddress.Line1, order.BillingAddress.Line2,
		order.BillingAddress.City, order.BillingAddress.Region, order.BillingAddress.PostalCode, order.BillingAddress.CountryCode,
		order.DeliveryMethod, order.DeliveryFee.AmountMinor, order.DeliveryFee.Currency, order.PaymentMethod, order.DeliveryOfficeID, order.ReservationID, order.CartGuestToken,
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
		var providerOrderID *string
		if order.Payment.ProviderOrderID != "" {
			providerOrderID = &order.Payment.ProviderOrderID
		}
		paymentRow := tx.QueryRow(ctx, `
			INSERT INTO order_payments (order_id, provider, provider_order_id, provider_reference, status, amount_minor, currency)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING `+paymentColumns,
			created.ID, order.Payment.Provider, providerOrderID, order.Payment.ProviderReference, order.Payment.Status,
			order.Payment.Amount.AmountMinor, order.Payment.Amount.Currency)

		payment, err := scanOrderPayment(paymentRow)
		if err != nil {
			return nil, err
		}
		created.Payment = payment

		// Ledger: opening a payment (Revolut order created) is the first audit
		// entry. Only card orders carry a Payment; pay-on-delivery orders don't.
		if err := insertPaymentTransaction(ctx, tx, created.ID, order.Payment.Provider,
			order.Payment.ProviderOrderID, order.Payment.ProviderReference,
			domain.PaymentTxnInitiated, order.Payment.Status,
			order.Payment.Amount.AmountMinor, order.Payment.Amount.Currency); err != nil {
			return nil, err
		}
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
	row := r.db.QueryRow(ctx, `SELECT `+paymentColumns+`
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

// FindByProviderOrderID resolves the order behind a Revolut order id via its
// payment row, then loads the full order.
func (r *PostgresRepository) FindByProviderOrderID(ctx context.Context, providerOrderID string) (*domain.Order, error) {
	var orderID uuid.UUID
	err := r.db.QueryRow(ctx,
		`SELECT order_id FROM order_payments WHERE provider_order_id = $1 ORDER BY created_at DESC LIMIT 1`,
		providerOrderID).Scan(&orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, orderID)
}

// MarkPaid settles a card order: only a pending_payment order transitions, so
// a duplicated/late webhook is a harmless no-op that appends no ledger row.
func (r *PostgresRepository) MarkPaid(ctx context.Context, orderID uuid.UUID, providerReference string, capturedMinor int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var settled uuid.UUID
	err = tx.QueryRow(ctx,
		`UPDATE orders SET status = 'paid', updated_at = NOW() WHERE id = $1 AND status = 'pending_payment' RETURNING id`,
		orderID).Scan(&settled)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // already settled — no-op, no duplicate ledger entry
	}
	if err != nil {
		return err
	}

	var provider, providerOrderID, currency string
	if err := tx.QueryRow(ctx,
		`UPDATE order_payments SET status = 'succeeded', provider_reference = $2, captured_minor = $3, updated_at = NOW()
		 WHERE order_id = $1 RETURNING provider, COALESCE(provider_order_id, ''), currency`,
		orderID, providerReference, capturedMinor).Scan(&provider, &providerOrderID, &currency); err != nil {
		return err
	}
	if err := insertPaymentTransaction(ctx, tx, orderID, provider, providerOrderID, providerReference,
		domain.PaymentTxnCaptured, "succeeded", capturedMinor, currency); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// MarkPaymentFailed moves a pending_payment order to payment_failed. Idempotent
// like MarkPaid: no transition means no ledger row.
func (r *PostgresRepository) MarkPaymentFailed(ctx context.Context, orderID uuid.UUID, reason string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var failed uuid.UUID
	err = tx.QueryRow(ctx,
		`UPDATE orders SET status = 'payment_failed', updated_at = NOW() WHERE id = $1 AND status = 'pending_payment' RETURNING id`,
		orderID).Scan(&failed)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	var provider, providerOrderID, currency string
	var amountMinor int64
	if err := tx.QueryRow(ctx,
		`UPDATE order_payments SET status = 'failed', updated_at = NOW()
		 WHERE order_id = $1 RETURNING provider, COALESCE(provider_order_id, ''), amount_minor, currency`,
		orderID).Scan(&provider, &providerOrderID, &amountMinor, &currency); err != nil {
		return err
	}
	// The failure reason (e.g. "ORDER_CANCELLED", "abandoned") is the most
	// useful audit descriptor here; the full webhook payload is also retained in
	// payment_webhook_events. A failed payment has no settlement reference.
	failStatus := reason
	if failStatus == "" {
		failStatus = "failed"
	}
	if err := insertPaymentTransaction(ctx, tx, orderID, provider, providerOrderID, "",
		domain.PaymentTxnFailed, failStatus, amountMinor, currency); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PostgresRepository) GetOrderPaymentContext(ctx context.Context, orderID uuid.UUID) (application.OrderPaymentContext, error) {
	var pc application.OrderPaymentContext
	err := r.db.QueryRow(ctx, `
		SELECT o.status, COALESCE(p.provider_order_id, ''), COALESCE(p.captured_minor, 0), COALESCE(p.refunded_minor, 0), o.total_currency
		FROM orders o
		LEFT JOIN order_payments p ON p.order_id = o.id
		WHERE o.id = $1
		ORDER BY p.created_at DESC NULLS LAST
		LIMIT 1`, orderID,
	).Scan(&pc.Status, &pc.ProviderOrderID, &pc.CapturedMinor, &pc.RefundedMinor, &pc.Currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.OrderPaymentContext{}, domain.ErrOrderNotFound
	}
	if err != nil {
		return application.OrderPaymentContext{}, err
	}
	return pc, nil
}

// RecordRefund inserts a refund and, when it's completed, advances the
// payment's refunded total and the order's rolled-up status — one transaction.
func (r *PostgresRepository) RecordRefund(ctx context.Context, input application.RecordRefundInput) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var providerRefundID *string
	if input.ProviderRefundID != "" {
		providerRefundID = &input.ProviderRefundID
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO order_refunds (order_id, provider_refund_id, amount_minor, currency, reason, state, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		input.OrderID, providerRefundID, input.AmountMinor, input.Currency, nullifyEmpty(input.Reason), input.State, input.CreatedBy,
	); err != nil {
		return err
	}

	// Ledger: record the refund event. provider comes from the order's payment
	// row so the entry ties back to the same Revolut order as the capture.
	var provider, providerOrderID string
	if err := tx.QueryRow(ctx,
		`SELECT provider, COALESCE(provider_order_id, '') FROM order_payments WHERE order_id = $1 ORDER BY created_at DESC LIMIT 1`,
		input.OrderID).Scan(&provider, &providerOrderID); err != nil {
		return err
	}
	if err := insertPaymentTransaction(ctx, tx, input.OrderID, provider, providerOrderID, input.ProviderRefundID,
		domain.PaymentTxnRefunded, input.State, input.AmountMinor, input.Currency); err != nil {
		return err
	}

	if input.State == "completed" {
		if _, err := tx.Exec(ctx,
			`UPDATE order_payments SET refunded_minor = refunded_minor + $2, updated_at = NOW() WHERE order_id = $1`,
			input.OrderID, input.AmountMinor); err != nil {
			return err
		}
		if input.OrderStatus != "" {
			if _, err := tx.Exec(ctx,
				`UPDATE orders SET status = $2, updated_at = NOW() WHERE id = $1`,
				input.OrderID, input.OrderStatus); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

func nullifyEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// insertPaymentTransaction appends one immutable row to the payment audit
// ledger. Always called with the enclosing tx so the entry commits atomically
// with the state change that produced it.
func insertPaymentTransaction(ctx context.Context, tx pgx.Tx, orderID uuid.UUID, provider, providerOrderID, providerReference, txType, status string, amountMinor int64, currency string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO payment_transactions (order_id, provider, provider_order_id, provider_reference, type, status, amount_minor, currency)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		orderID, provider, nullifyEmpty(providerOrderID), nullifyEmpty(providerReference), txType, nullifyEmpty(status), amountMinor, currency)
	return err
}

// ListPaymentTransactions returns an order's payment audit trail, oldest first.
func (r *PostgresRepository) ListPaymentTransactions(ctx context.Context, orderID uuid.UUID) ([]domain.PaymentTransaction, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, order_id, provider, COALESCE(provider_order_id, ''), COALESCE(provider_reference, ''),
		       type, COALESCE(status, ''), amount_minor, currency, created_at
		FROM payment_transactions WHERE order_id = $1 ORDER BY created_at`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	txns := []domain.PaymentTransaction{}
	for rows.Next() {
		var t domain.PaymentTransaction
		var amount int64
		var currency string
		if err := rows.Scan(&t.ID, &t.OrderID, &t.Provider, &t.ProviderOrderID, &t.ProviderReference,
			&t.Type, &t.Status, &amount, &currency, &t.CreatedAt); err != nil {
			return nil, err
		}
		t.Amount = money.Money{AmountMinor: amount, Currency: currency}
		txns = append(txns, t)
	}
	return txns, rows.Err()
}

func (r *PostgresRepository) ListPendingPaymentOlderThan(ctx context.Context, cutoff time.Time) ([]application.PendingPaymentRef, error) {
	rows, err := r.db.Query(ctx, `
		SELECT o.id, p.provider_order_id
		FROM orders o
		JOIN order_payments p ON p.order_id = o.id
		WHERE o.status = 'pending_payment'
		  AND p.provider_order_id IS NOT NULL
		  AND o.created_at < $1`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	refs := []application.PendingPaymentRef{}
	for rows.Next() {
		var ref application.PendingPaymentRef
		if err := rows.Scan(&ref.OrderID, &ref.ProviderOrderID); err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, rows.Err()
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
		&o.Carrier, &o.TrackingNumber, &o.ShipmentStatus, &o.SpeedyShipmentID, &o.DeliveryOfficeID, &o.ViewedByAdminAt, &o.ReservationID, &o.CartGuestToken,
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
	if err := row.Scan(&p.ID, &p.OrderID, &p.Provider, &p.ProviderOrderID, &p.ProviderReference,
		&p.Status, &amount, &currency, &p.CapturedMinor, &p.RefundedMinor, &p.CreatedAt); err != nil {
		return nil, err
	}
	p.Amount = money.Money{AmountMinor: amount, Currency: currency}
	return &p, nil
}
