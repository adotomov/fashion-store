package infrastructure

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

const variantsForQuery = `
	SELECT v.id, v.product_id, v.price_override_amount, v.price_override_currency, v.created_at, v.updated_at,
	       ii.id, ii.quantity_on_hand, ii.quantity_reserved
	FROM product_variants v
	LEFT JOIN inventory_items ii ON ii.variant_id = v.id
	WHERE v.product_id = $1 ORDER BY v.created_at`

func (r *PostgresProductRepository) variantsFor(ctx context.Context, productID uuid.UUID) ([]domain.ProductVariant, error) {
	rows, err := r.db.Query(ctx, variantsForQuery, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var variants []domain.ProductVariant
	for rows.Next() {
		v, err := scanVariantWithInventory(rows)
		if err != nil {
			return nil, err
		}
		variants = append(variants, *v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range variants {
		attrs, err := r.attributeValuesFor(ctx, "variant_attribute_values", "variant_id", variants[i].ID)
		if err != nil {
			return nil, err
		}
		variants[i].Attributes = attrs
	}

	return variants, nil
}

func (r *PostgresProductRepository) CreateVariant(ctx context.Context, variant domain.ProductVariant, attributeValueIDs []uuid.UUID) (*domain.ProductVariant, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var amount, currency any
	if variant.PriceOverride != nil {
		amount, currency = variant.PriceOverride.AmountMinor, variant.PriceOverride.Currency
	}

	row := tx.QueryRow(ctx, `
		INSERT INTO product_variants (product_id, price_override_amount, price_override_currency)
		VALUES ($1, $2, $3)
		RETURNING id, product_id, price_override_amount, price_override_currency, created_at, updated_at`,
		variant.ProductID, amount, currency)

	created, err := scanVariant(row)
	if err != nil {
		return nil, err
	}

	for _, valueID := range attributeValueIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO variant_attribute_values (variant_id, attribute_value_id) VALUES ($1, $2)`,
			created.ID, valueID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	created.Attributes, err = r.attributeValuesFor(ctx, "variant_attribute_values", "variant_id", created.ID)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *PostgresProductRepository) FindVariantByID(ctx context.Context, variantID uuid.UUID) (*domain.ProductVariant, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, product_id, price_override_amount, price_override_currency, created_at, updated_at
		FROM product_variants WHERE id = $1`, variantID)

	variant, err := scanVariant(row)
	if err != nil {
		return nil, err
	}

	variant.Attributes, err = r.attributeValuesFor(ctx, "variant_attribute_values", "variant_id", variant.ID)
	if err != nil {
		return nil, err
	}

	return variant, nil
}

func (r *PostgresProductRepository) UpdateVariant(ctx context.Context, variant domain.ProductVariant, attributeValueIDs []uuid.UUID) (*domain.ProductVariant, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var amount, currency any
	if variant.PriceOverride != nil {
		amount, currency = variant.PriceOverride.AmountMinor, variant.PriceOverride.Currency
	}

	row := tx.QueryRow(ctx, `
		UPDATE product_variants SET price_override_amount = $2, price_override_currency = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING id, product_id, price_override_amount, price_override_currency, created_at, updated_at`,
		variant.ID, amount, currency)

	updated, err := scanVariant(row)
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM variant_attribute_values WHERE variant_id = $1`, variant.ID); err != nil {
		return nil, err
	}
	for _, valueID := range attributeValueIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO variant_attribute_values (variant_id, attribute_value_id) VALUES ($1, $2)`,
			variant.ID, valueID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	updated.Attributes, err = r.attributeValuesFor(ctx, "variant_attribute_values", "variant_id", updated.ID)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (r *PostgresProductRepository) DeleteVariant(ctx context.Context, variantID uuid.UUID) error {
	// Cascades to variant_attribute_values and inventory_items (and
	// transitively inventory_movements) via ON DELETE CASCADE. Deleting a
	// variant with inventory history is an edge case; acceptable for MVP.
	tag, err := r.db.Exec(ctx, `DELETE FROM product_variants WHERE id = $1`, variantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrVariantNotFound
	}
	return nil
}

func scanVariant(row pgx.Row) (*domain.ProductVariant, error) {
	var v domain.ProductVariant
	var amount *int64
	var currency *string
	err := row.Scan(&v.ID, &v.ProductID, &amount, &currency, &v.CreatedAt, &v.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrVariantNotFound
	}
	if err != nil {
		return nil, err
	}
	if amount != nil && currency != nil {
		v.PriceOverride = &money.Money{AmountMinor: *amount, Currency: *currency}
	}
	return &v, nil
}

func scanVariantWithInventory(row pgx.Row) (*domain.ProductVariant, error) {
	var v domain.ProductVariant
	var amount *int64
	var currency *string
	var inventoryItemID *uuid.UUID
	var onHand, reserved *int
	err := row.Scan(&v.ID, &v.ProductID, &amount, &currency, &v.CreatedAt, &v.UpdatedAt, &inventoryItemID, &onHand, &reserved)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrVariantNotFound
	}
	if err != nil {
		return nil, err
	}
	if amount != nil && currency != nil {
		v.PriceOverride = &money.Money{AmountMinor: *amount, Currency: *currency}
	}
	if inventoryItemID != nil {
		v.InventoryItemID = inventoryItemID
		available := 0
		if onHand != nil {
			available += *onHand
		}
		if reserved != nil {
			available -= *reserved
		}
		v.QuantityAvailable = &available
	}
	return &v, nil
}

func (r *PostgresProductRepository) mediaFor(ctx context.Context, productID uuid.UUID) ([]domain.ProductMedia, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, product_id, bucket, object_key, COALESCE(content_type, ''), COALESCE(size_bytes, 0), position, COALESCE(alt_text, ''), created_at
		FROM product_media WHERE product_id = $1 ORDER BY position`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	media := []domain.ProductMedia{}
	for rows.Next() {
		m, err := scanMedia(rows)
		if err != nil {
			return nil, err
		}
		media = append(media, *m)
	}
	return media, rows.Err()
}

func (r *PostgresProductRepository) CreateMedia(ctx context.Context, media domain.ProductMedia) (*domain.ProductMedia, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO product_media (product_id, bucket, object_key, content_type, size_bytes, position, alt_text)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, product_id, bucket, object_key, COALESCE(content_type, ''), COALESCE(size_bytes, 0), position, COALESCE(alt_text, ''), created_at`,
		media.ProductID, media.Bucket, media.ObjectKey, media.ContentType, media.SizeBytes, media.Position, media.AltText)

	return scanMedia(row)
}

func (r *PostgresProductRepository) FindMediaByID(ctx context.Context, mediaID uuid.UUID) (*domain.ProductMedia, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, product_id, bucket, object_key, COALESCE(content_type, ''), COALESCE(size_bytes, 0), position, COALESCE(alt_text, ''), created_at
		FROM product_media WHERE id = $1`, mediaID)

	return scanMedia(row)
}

func (r *PostgresProductRepository) UpdateMedia(ctx context.Context, media domain.ProductMedia) (*domain.ProductMedia, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE product_media SET position = $2, alt_text = $3
		WHERE id = $1
		RETURNING id, product_id, bucket, object_key, COALESCE(content_type, ''), COALESCE(size_bytes, 0), position, COALESCE(alt_text, ''), created_at`,
		media.ID, media.Position, media.AltText)

	return scanMedia(row)
}

func (r *PostgresProductRepository) DeleteMedia(ctx context.Context, mediaID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM product_media WHERE id = $1`, mediaID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrMediaNotFound
	}
	return nil
}

func scanMedia(row pgx.Row) (*domain.ProductMedia, error) {
	var m domain.ProductMedia
	err := row.Scan(&m.ID, &m.ProductID, &m.Bucket, &m.ObjectKey, &m.ContentType, &m.SizeBytes, &m.Position, &m.AltText, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrMediaNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}
