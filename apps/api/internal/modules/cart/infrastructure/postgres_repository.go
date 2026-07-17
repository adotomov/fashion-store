package infrastructure

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/cart/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

// untrackedAvailability is used for variants with no inventory_items row —
// inventory tracking is opt-in per variant, so "not tracked" must mean
// "no known limit," not "zero stock."
const untrackedAvailability = math.MaxInt32

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const cartItemsQuery = `
	SELECT ci.id, ci.variant_id, p.id, p.name, p.slug,
	       p.base_price_amount, p.base_price_currency,
	       v.price_override_amount, v.price_override_currency,
	       (SELECT m.id FROM product_media m WHERE m.product_id = p.id ORDER BY m.position LIMIT 1),
	       (ii.id IS NOT NULL), COALESCE(ii.quantity_on_hand - ii.quantity_reserved, 0),
	       COALESCE((SELECT string_agg(av.value, ' / ' ORDER BY av.value)
	                 FROM variant_attribute_values vav
	                 JOIN attribute_values av ON av.id = vav.attribute_value_id
	                 WHERE vav.variant_id = v.id), ''),
	       ci.quantity, ci.created_at, ci.updated_at
	FROM cart_items ci
	JOIN product_variants v ON v.id = ci.variant_id
	JOIN products p ON p.id = v.product_id
	LEFT JOIN inventory_items ii ON ii.variant_id = v.id
	WHERE ci.cart_id = $1
	ORDER BY ci.created_at`

func (r *PostgresRepository) loadItems(ctx context.Context, cartID uuid.UUID) ([]domain.CartItem, error) {
	rows, err := r.db.Query(ctx, cartItemsQuery, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.CartItem{}
	for rows.Next() {
		var item domain.CartItem
		var baseAmount int64
		var baseCurrency string
		var overrideAmount *int64
		var overrideCurrency *string
		var tracked bool
		item.CartID = cartID

		if err := rows.Scan(
			&item.ID, &item.VariantID, &item.ProductID, &item.ProductName, &item.ProductSlug,
			&baseAmount, &baseCurrency, &overrideAmount, &overrideCurrency,
			&item.ImageMediaID, &tracked, &item.AvailableQuantity, &item.VariantLabel,
			&item.Quantity, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if !tracked {
			item.AvailableQuantity = untrackedAvailability
		}

		item.UnitPrice = money.Money{AmountMinor: baseAmount, Currency: baseCurrency}
		if overrideAmount != nil && overrideCurrency != nil {
			item.UnitPrice = money.Money{AmountMinor: *overrideAmount, Currency: *overrideCurrency}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) FindByID(ctx context.Context, cartID uuid.UUID) (*domain.Cart, error) {
	row := r.db.QueryRow(ctx, `SELECT id, user_id, guest_token, created_at, updated_at FROM carts WHERE id = $1`, cartID)
	return r.scanCartWithItems(ctx, row)
}

func (r *PostgresRepository) FindByUser(ctx context.Context, userID uuid.UUID) (*domain.Cart, error) {
	row := r.db.QueryRow(ctx, `SELECT id, user_id, guest_token, created_at, updated_at FROM carts WHERE user_id = $1`, userID)
	return r.scanCartWithItems(ctx, row)
}

func (r *PostgresRepository) FindByGuestToken(ctx context.Context, token uuid.UUID) (*domain.Cart, error) {
	row := r.db.QueryRow(ctx, `SELECT id, user_id, guest_token, created_at, updated_at FROM carts WHERE guest_token = $1`, token)
	return r.scanCartWithItems(ctx, row)
}

func (r *PostgresRepository) scanCartWithItems(ctx context.Context, row pgx.Row) (*domain.Cart, error) {
	var cart domain.Cart
	err := row.Scan(&cart.ID, &cart.UserID, &cart.GuestToken, &cart.CreatedAt, &cart.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrCartNotFound
	}
	if err != nil {
		return nil, err
	}

	items, err := r.loadItems(ctx, cart.ID)
	if err != nil {
		return nil, err
	}
	cart.Items = items
	return &cart, nil
}

func (r *PostgresRepository) CreateForUser(ctx context.Context, userID uuid.UUID) (*domain.Cart, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO carts (user_id) VALUES ($1)
		RETURNING id, user_id, guest_token, created_at, updated_at`, userID)
	return r.scanCartWithItems(ctx, row)
}

func (r *PostgresRepository) CreateForGuest(ctx context.Context, token uuid.UUID) (*domain.Cart, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO carts (guest_token) VALUES ($1)
		RETURNING id, user_id, guest_token, created_at, updated_at`, token)
	return r.scanCartWithItems(ctx, row)
}

func (r *PostgresRepository) availableForVariant(ctx context.Context, variantID uuid.UUID) (int, error) {
	var tracked bool
	var available int
	err := r.db.QueryRow(ctx, `
		SELECT (ii.id IS NOT NULL), COALESCE(ii.quantity_on_hand - ii.quantity_reserved, 0)
		FROM product_variants v
		LEFT JOIN inventory_items ii ON ii.variant_id = v.id
		WHERE v.id = $1`, variantID).Scan(&tracked, &available)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, domain.ErrVariantNotFound
	}
	if err != nil {
		return 0, err
	}
	if !tracked {
		return untrackedAvailability, nil
	}
	return available, nil
}

func (r *PostgresRepository) AddOrIncrementItem(ctx context.Context, cartID, variantID uuid.UUID, quantity int) (*domain.Cart, error) {
	available, err := r.availableForVariant(ctx, variantID)
	if err != nil {
		return nil, err
	}

	var existingQty int
	err = r.db.QueryRow(ctx, `SELECT quantity FROM cart_items WHERE cart_id = $1 AND variant_id = $2`, cartID, variantID).Scan(&existingQty)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	if existingQty+quantity > available {
		return nil, domain.ErrInsufficientStock
	}

	if _, err := r.db.Exec(ctx, `
		INSERT INTO cart_items (cart_id, variant_id, quantity)
		VALUES ($1, $2, $3)
		ON CONFLICT (cart_id, variant_id) DO UPDATE SET quantity = cart_items.quantity + EXCLUDED.quantity, updated_at = NOW()`,
		cartID, variantID, quantity); err != nil {
		return nil, err
	}
	if _, err := r.db.Exec(ctx, `UPDATE carts SET updated_at = NOW() WHERE id = $1`, cartID); err != nil {
		return nil, err
	}
	return r.FindByID(ctx, cartID)
}

func (r *PostgresRepository) SetItemQuantity(ctx context.Context, cartID, itemID uuid.UUID, quantity int) (*domain.Cart, error) {
	var variantID uuid.UUID
	err := r.db.QueryRow(ctx, `SELECT variant_id FROM cart_items WHERE id = $1 AND cart_id = $2`, itemID, cartID).Scan(&variantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrCartItemNotFound
	}
	if err != nil {
		return nil, err
	}

	available, err := r.availableForVariant(ctx, variantID)
	if err != nil {
		return nil, err
	}
	if quantity > available {
		return nil, domain.ErrInsufficientStock
	}

	tag, err := r.db.Exec(ctx, `UPDATE cart_items SET quantity = $1, updated_at = NOW() WHERE id = $2 AND cart_id = $3`, quantity, itemID, cartID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, domain.ErrCartItemNotFound
	}
	if _, err := r.db.Exec(ctx, `UPDATE carts SET updated_at = NOW() WHERE id = $1`, cartID); err != nil {
		return nil, err
	}
	return r.FindByID(ctx, cartID)
}

func (r *PostgresRepository) RemoveItem(ctx context.Context, cartID, itemID uuid.UUID) (*domain.Cart, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM cart_items WHERE id = $1 AND cart_id = $2`, itemID, cartID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, domain.ErrCartItemNotFound
	}
	if _, err := r.db.Exec(ctx, `UPDATE carts SET updated_at = NOW() WHERE id = $1`, cartID); err != nil {
		return nil, err
	}
	return r.FindByID(ctx, cartID)
}

func (r *PostgresRepository) ClearItems(ctx context.Context, cartID uuid.UUID) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM cart_items WHERE cart_id = $1`, cartID); err != nil {
		return err
	}
	_, err := r.db.Exec(ctx, `UPDATE carts SET updated_at = NOW() WHERE id = $1`, cartID)
	return err
}

// SetReservation records (or replaces) the checkout hold on a cart.
func (r *PostgresRepository) SetReservation(ctx context.Context, cartID, reservationID uuid.UUID, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE carts SET reservation_id = $2, reservation_expires_at = $3, updated_at = NOW() WHERE id = $1`,
		cartID, reservationID, expiresAt)
	return err
}

// GetReservation returns the cart's current hold, or (nil, nil, nil) if none.
func (r *PostgresRepository) GetReservation(ctx context.Context, cartID uuid.UUID) (*uuid.UUID, *time.Time, error) {
	var reservationID *uuid.UUID
	var expiresAt *time.Time
	err := r.db.QueryRow(ctx, `SELECT reservation_id, reservation_expires_at FROM carts WHERE id = $1`, cartID).
		Scan(&reservationID, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, domain.ErrCartNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	return reservationID, expiresAt, nil
}

// ClearReservation drops the checkout hold columns (the reservation itself is
// committed/released by the caller before or after this).
func (r *PostgresRepository) ClearReservation(ctx context.Context, cartID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE carts SET reservation_id = NULL, reservation_expires_at = NULL, updated_at = NOW() WHERE id = $1`,
		cartID)
	return err
}

// ListExpiredReservations returns holds whose expiry has passed, so the sweeper
// can release the stock and clear the columns.
func (r *PostgresRepository) ListExpiredReservations(ctx context.Context, cutoff time.Time) ([]domain.ExpiredReservation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT user_id, guest_token, reservation_id FROM carts
		WHERE reservation_id IS NOT NULL AND reservation_expires_at < $1`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.ExpiredReservation{}
	for rows.Next() {
		var e domain.ExpiredReservation
		if err := rows.Scan(&e.UserID, &e.GuestToken, &e.ReservationID); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) MergeCarts(ctx context.Context, sourceCartID, targetCartID uuid.UUID) (*domain.Cart, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO cart_items (cart_id, variant_id, quantity)
		SELECT $1, variant_id, quantity FROM cart_items WHERE cart_id = $2
		ON CONFLICT (cart_id, variant_id) DO UPDATE SET quantity = cart_items.quantity + EXCLUDED.quantity, updated_at = NOW()`,
		targetCartID, sourceCartID); err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM carts WHERE id = $1`, sourceCartID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE carts SET updated_at = NOW() WHERE id = $1`, targetCartID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.FindByID(ctx, targetCartID)
}
