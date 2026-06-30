package infrastructure

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/wishlist/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// wishlistItemsBaseQuery deliberately omits ORDER BY so callers can append
// additional WHERE conditions (e.g. Add's "AND wi.product_id = $2") without
// producing invalid SQL — List appends its own ORDER BY after the WHERE is
// otherwise complete.
const wishlistItemsBaseQuery = `
	SELECT wi.id, wi.product_id, p.name, p.slug,
	       p.base_price_amount, p.base_price_currency,
	       p.compare_at_price_amount, p.compare_at_price_currency,
	       (SELECT m.id FROM product_media m WHERE m.product_id = p.id ORDER BY m.position LIMIT 1),
	       NOT EXISTS (SELECT 1 FROM product_variants v WHERE v.product_id = p.id)
	         OR EXISTS (
	           SELECT 1 FROM product_variants v
	           JOIN inventory_items ii ON ii.variant_id = v.id
	           WHERE v.product_id = p.id AND (ii.quantity_on_hand - ii.quantity_reserved) > 0
	         ) AS in_stock,
	       COALESCE((
	         SELECT array_agg(DISTINCT av.value ORDER BY av.value)
	         FROM product_variants v
	         JOIN variant_attribute_values vav ON vav.variant_id = v.id
	         JOIN attribute_values av ON av.id = vav.attribute_value_id
	         JOIN attributes attr ON attr.id = av.attribute_id
	         WHERE v.product_id = p.id AND attr.name ILIKE 'size'
	       ), ARRAY[]::text[]) AS sizes,
	       wi.created_at
	FROM wishlist_items wi
	JOIN products p ON p.id = wi.product_id
	WHERE wi.user_id = $1`

const wishlistItemsQuery = wishlistItemsBaseQuery + "\n\tORDER BY wi.created_at DESC"

func scanItem(row pgx.Row) (*domain.Item, error) {
	var item domain.Item
	var baseAmount int64
	var baseCurrency string
	var compareAmount *int64
	var compareCurrency *string

	if err := row.Scan(
		&item.ID, &item.ProductID, &item.ProductName, &item.ProductSlug,
		&baseAmount, &baseCurrency, &compareAmount, &compareCurrency,
		&item.ImageMediaID, &item.InStock, &item.Sizes, &item.CreatedAt,
	); err != nil {
		return nil, err
	}

	item.BasePrice = money.Money{AmountMinor: baseAmount, Currency: baseCurrency}
	if compareAmount != nil && compareCurrency != nil {
		compareAt := money.Money{AmountMinor: *compareAmount, Currency: *compareCurrency}
		item.CompareAtPrice = &compareAt
	}
	return &item, nil
}

func (r *PostgresRepository) List(ctx context.Context, userID uuid.UUID) ([]domain.Item, error) {
	rows, err := r.db.Query(ctx, wishlistItemsQuery, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.Item{}
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) Add(ctx context.Context, userID, productID uuid.UUID) (*domain.Item, error) {
	_, err := r.db.Exec(ctx, `
		INSERT INTO wishlist_items (user_id, product_id) VALUES ($1, $2)
		ON CONFLICT (user_id, product_id) DO NOTHING`, userID, productID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return nil, domain.ErrProductNotFound
		}
		return nil, err
	}

	row := r.db.QueryRow(ctx, wishlistItemsBaseQuery+" AND wi.product_id = $2", userID, productID)
	return scanItem(row)
}

func (r *PostgresRepository) Remove(ctx context.Context, userID, productID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM wishlist_items WHERE user_id = $1 AND product_id = $2`, userID, productID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrItemNotFound
	}
	return nil
}
