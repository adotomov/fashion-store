package infrastructure

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/promotions/domain"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const promotionCols = `id, name, COALESCE(description,''), type,
	value_percent, value_fixed_minor, value_fixed_currency,
	buy_qty, get_qty, get_discount_pct,
	min_quantity, target_type,
	starts_at, ends_at, is_active, priority,
	created_at, updated_at`

func scanPromotion(row pgx.Row) (*domain.Promotion, error) {
	var p domain.Promotion
	err := row.Scan(
		&p.ID, &p.Name, &p.Description, &p.Type,
		&p.ValuePercent, &p.ValueFixedMinor, &p.ValueFixedCurrency,
		&p.BuyQty, &p.GetQty, &p.GetDiscountPct,
		&p.MinQuantity, &p.TargetType,
		&p.StartsAt, &p.EndsAt, &p.IsActive, &p.Priority,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPromotionNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *PostgresRepository) ListPromotions(ctx context.Context) ([]domain.Promotion, error) {
	rows, err := r.db.Query(ctx, `SELECT `+promotionCols+` FROM promotions ORDER BY priority DESC, created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Promotion
	for rows.Next() {
		p, err := scanPromotion(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := r.loadTargets(ctx, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *PostgresRepository) GetPromotion(ctx context.Context, id uuid.UUID) (*domain.Promotion, error) {
	p, err := scanPromotion(r.db.QueryRow(ctx, `SELECT `+promotionCols+` FROM promotions WHERE id = $1`, id))
	if err != nil {
		return nil, err
	}
	list := []domain.Promotion{*p}
	if err := r.loadTargets(ctx, list); err != nil {
		return nil, err
	}
	return &list[0], nil
}

func (r *PostgresRepository) CreatePromotion(ctx context.Context, p domain.Promotion) (*domain.Promotion, error) {
	return scanPromotion(r.db.QueryRow(ctx, `
		INSERT INTO promotions (name, description, type,
			value_percent, value_fixed_minor, value_fixed_currency,
			buy_qty, get_qty, get_discount_pct,
			min_quantity, target_type,
			starts_at, ends_at, is_active, priority)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING `+promotionCols,
		p.Name, p.Description, p.Type,
		p.ValuePercent, p.ValueFixedMinor, p.ValueFixedCurrency,
		p.BuyQty, p.GetQty, p.GetDiscountPct,
		p.MinQuantity, p.TargetType,
		p.StartsAt, p.EndsAt, p.IsActive, p.Priority,
	))
}

func (r *PostgresRepository) UpdatePromotion(ctx context.Context, p domain.Promotion) (*domain.Promotion, error) {
	updated, err := scanPromotion(r.db.QueryRow(ctx, `
		UPDATE promotions SET
			name=$2, description=$3, type=$4,
			value_percent=$5, value_fixed_minor=$6, value_fixed_currency=$7,
			buy_qty=$8, get_qty=$9, get_discount_pct=$10,
			min_quantity=$11, target_type=$12,
			starts_at=$13, ends_at=$14, is_active=$15, priority=$16,
			updated_at=NOW()
		WHERE id=$1
		RETURNING `+promotionCols,
		p.ID, p.Name, p.Description, p.Type,
		p.ValuePercent, p.ValueFixedMinor, p.ValueFixedCurrency,
		p.BuyQty, p.GetQty, p.GetDiscountPct,
		p.MinQuantity, p.TargetType,
		p.StartsAt, p.EndsAt, p.IsActive, p.Priority,
	))
	return updated, err
}

func (r *PostgresRepository) DeletePromotion(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM promotions WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrPromotionNotFound
	}
	return nil
}

func (r *PostgresRepository) SetPromotionCategories(ctx context.Context, promotionID uuid.UUID, categoryIDs []uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM promotion_categories WHERE promotion_id=$1`, promotionID)
	if err != nil {
		return err
	}
	for _, id := range categoryIDs {
		if _, err := r.db.Exec(ctx, `INSERT INTO promotion_categories VALUES ($1,$2) ON CONFLICT DO NOTHING`, promotionID, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepository) SetPromotionProductTypes(ctx context.Context, promotionID uuid.UUID, typeIDs []uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM promotion_product_types WHERE promotion_id=$1`, promotionID)
	if err != nil {
		return err
	}
	for _, id := range typeIDs {
		if _, err := r.db.Exec(ctx, `INSERT INTO promotion_product_types VALUES ($1,$2) ON CONFLICT DO NOTHING`, promotionID, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepository) SetPromotionProducts(ctx context.Context, promotionID uuid.UUID, productIDs []uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM promotion_products WHERE promotion_id=$1`, promotionID)
	if err != nil {
		return err
	}
	for _, id := range productIDs {
		if _, err := r.db.Exec(ctx, `INSERT INTO promotion_products VALUES ($1,$2) ON CONFLICT DO NOTHING`, promotionID, id); err != nil {
			return err
		}
	}
	return nil
}

// loadTargets bulk-loads category/type/product associations for a slice of promotions.
func (r *PostgresRepository) loadTargets(ctx context.Context, promotions []domain.Promotion) error {
	if len(promotions) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(promotions))
	idx := make(map[uuid.UUID]int, len(promotions))
	for i, p := range promotions {
		ids[i] = p.ID
		idx[p.ID] = i
	}

	catRows, err := r.db.Query(ctx, `SELECT promotion_id, category_id FROM promotion_categories WHERE promotion_id = ANY($1)`, ids)
	if err != nil {
		return err
	}
	defer catRows.Close()
	for catRows.Next() {
		var pid, cid uuid.UUID
		if err := catRows.Scan(&pid, &cid); err != nil {
			return err
		}
		promotions[idx[pid]].CategoryIDs = append(promotions[idx[pid]].CategoryIDs, cid)
	}
	if err := catRows.Err(); err != nil {
		return err
	}

	typeRows, err := r.db.Query(ctx, `SELECT promotion_id, product_type_id FROM promotion_product_types WHERE promotion_id = ANY($1)`, ids)
	if err != nil {
		return err
	}
	defer typeRows.Close()
	for typeRows.Next() {
		var pid, tid uuid.UUID
		if err := typeRows.Scan(&pid, &tid); err != nil {
			return err
		}
		promotions[idx[pid]].TypeIDs = append(promotions[idx[pid]].TypeIDs, tid)
	}
	if err := typeRows.Err(); err != nil {
		return err
	}

	prodRows, err := r.db.Query(ctx, `SELECT promotion_id, product_id FROM promotion_products WHERE promotion_id = ANY($1)`, ids)
	if err != nil {
		return err
	}
	defer prodRows.Close()
	for prodRows.Next() {
		var pid, oid uuid.UUID
		if err := prodRows.Scan(&pid, &oid); err != nil {
			return err
		}
		promotions[idx[pid]].ProductIDs = append(promotions[idx[pid]].ProductIDs, oid)
	}
	return prodRows.Err()
}

// GetEffectivePrices returns the highest-priority active promotion for each
// product in productIDs, matching through direct product, category, product-type,
// and "all" target types. A single CTE covers all four matching paths.
func (r *PostgresRepository) GetEffectivePrices(ctx context.Context, productIDs []uuid.UUID) (map[uuid.UUID]domain.Promotion, error) {
	if len(productIDs) == 0 {
		return nil, nil
	}

	rows, err := r.db.Query(ctx, `
	WITH applicable AS (
		-- Direct product target
		SELECT pp.product_id, pr.id as promo_id, pr.type, pr.value_percent,
		       pr.value_fixed_minor, pr.value_fixed_currency,
		       pr.buy_qty, pr.get_qty, pr.get_discount_pct,
		       pr.priority, pr.min_quantity
		FROM promotion_products pp
		JOIN promotions pr ON pr.id = pp.promotion_id
		WHERE pr.is_active = TRUE
		  AND (pr.starts_at IS NULL OR pr.starts_at <= NOW())
		  AND (pr.ends_at   IS NULL OR pr.ends_at   > NOW())
		  AND pp.product_id = ANY($1)

		UNION ALL

		-- Category target
		SELECT pc.product_id, pr.id, pr.type, pr.value_percent,
		       pr.value_fixed_minor, pr.value_fixed_currency,
		       pr.buy_qty, pr.get_qty, pr.get_discount_pct,
		       pr.priority, pr.min_quantity
		FROM product_categories pc
		JOIN promotion_categories pcat ON pcat.category_id = pc.category_id
		JOIN promotions pr ON pr.id = pcat.promotion_id
		WHERE pr.is_active = TRUE
		  AND (pr.starts_at IS NULL OR pr.starts_at <= NOW())
		  AND (pr.ends_at   IS NULL OR pr.ends_at   > NOW())
		  AND pc.product_id = ANY($1)

		UNION ALL

		-- Product-type target (join via categories)
		SELECT p2.id, pr.id, pr.type, pr.value_percent,
		       pr.value_fixed_minor, pr.value_fixed_currency,
		       pr.buy_qty, pr.get_qty, pr.get_discount_pct,
		       pr.priority, pr.min_quantity
		FROM products p2
		JOIN product_categories pc2 ON pc2.product_id = p2.id
		JOIN categories c ON c.id = pc2.category_id
		JOIN promotion_product_types ppt ON ppt.product_type_id = c.product_type_id
		JOIN promotions pr ON pr.id = ppt.promotion_id
		WHERE pr.is_active = TRUE
		  AND (pr.starts_at IS NULL OR pr.starts_at <= NOW())
		  AND (pr.ends_at   IS NULL OR pr.ends_at   > NOW())
		  AND p2.id = ANY($1)

		UNION ALL

		-- "All products" target
		SELECT p3.id, pr.id, pr.type, pr.value_percent,
		       pr.value_fixed_minor, pr.value_fixed_currency,
		       pr.buy_qty, pr.get_qty, pr.get_discount_pct,
		       pr.priority, pr.min_quantity
		FROM products p3
		JOIN promotions pr ON pr.target_type = 'all'
		WHERE pr.is_active = TRUE
		  AND (pr.starts_at IS NULL OR pr.starts_at <= NOW())
		  AND (pr.ends_at   IS NULL OR pr.ends_at   > NOW())
		  AND p3.id = ANY($1)
	),
	best AS (
		SELECT DISTINCT ON (product_id) *
		FROM applicable
		ORDER BY product_id, priority DESC,
		         COALESCE(value_percent, 0) DESC,
		         COALESCE(value_fixed_minor, 0) DESC
	)
	SELECT b.product_id,
	       pr.id, pr.name, COALESCE(pr.description,''), pr.type,
	       pr.value_percent, pr.value_fixed_minor, pr.value_fixed_currency,
	       pr.buy_qty, pr.get_qty, pr.get_discount_pct,
	       pr.min_quantity, pr.target_type,
	       pr.starts_at, pr.ends_at, pr.is_active, pr.priority,
	       pr.created_at, pr.updated_at
	FROM best b
	JOIN promotions pr ON pr.id = b.promo_id
	`, productIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID]domain.Promotion)
	for rows.Next() {
		var productID uuid.UUID
		var p domain.Promotion
		err := rows.Scan(
			&productID,
			&p.ID, &p.Name, &p.Description, &p.Type,
			&p.ValuePercent, &p.ValueFixedMinor, &p.ValueFixedCurrency,
			&p.BuyQty, &p.GetQty, &p.GetDiscountPct,
			&p.MinQuantity, &p.TargetType,
			&p.StartsAt, &p.EndsAt, &p.IsActive, &p.Priority,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		result[productID] = p
	}
	return result, rows.Err()
}

// --- Discount Codes ---

const codeCols = `id, code, value_percent, starts_at, expires_at, max_uses, use_count, is_active, created_at, updated_at`

func scanCode(row pgx.Row) (*domain.DiscountCode, error) {
	var c domain.DiscountCode
	err := row.Scan(&c.ID, &c.Code, &c.ValuePercent, &c.StartsAt, &c.ExpiresAt, &c.MaxUses, &c.UseCount, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCodeNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *PostgresRepository) ListCodes(ctx context.Context) ([]domain.DiscountCode, error) {
	rows, err := r.db.Query(ctx, `SELECT `+codeCols+` FROM discount_codes ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.DiscountCode
	for rows.Next() {
		c, err := scanCode(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *c)
	}
	return result, rows.Err()
}

func (r *PostgresRepository) GetCode(ctx context.Context, id uuid.UUID) (*domain.DiscountCode, error) {
	return scanCode(r.db.QueryRow(ctx, `SELECT `+codeCols+` FROM discount_codes WHERE id=$1`, id))
}

func (r *PostgresRepository) FindCode(ctx context.Context, code string) (*domain.DiscountCode, error) {
	return scanCode(r.db.QueryRow(ctx, `SELECT `+codeCols+` FROM discount_codes WHERE code=$1`, code))
}

func (r *PostgresRepository) CreateCode(ctx context.Context, c domain.DiscountCode) (*domain.DiscountCode, error) {
	created, err := scanCode(r.db.QueryRow(ctx, `
		INSERT INTO discount_codes (code, value_percent, starts_at, expires_at, max_uses, is_active)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING `+codeCols,
		c.Code, c.ValuePercent, c.StartsAt, c.ExpiresAt, c.MaxUses, c.IsActive,
	))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.ErrDuplicateCode
		}
		return nil, err
	}
	return created, nil
}

func (r *PostgresRepository) UpdateCode(ctx context.Context, c domain.DiscountCode) (*domain.DiscountCode, error) {
	return scanCode(r.db.QueryRow(ctx, `
		UPDATE discount_codes SET
			value_percent=$2, starts_at=$3, expires_at=$4,
			max_uses=$5, is_active=$6, updated_at=NOW()
		WHERE id=$1
		RETURNING `+codeCols,
		c.ID, c.ValuePercent, c.StartsAt, c.ExpiresAt, c.MaxUses, c.IsActive,
	))
}

func (r *PostgresRepository) DeleteCode(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM discount_codes WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrCodeNotFound
	}
	return nil
}

func (r *PostgresRepository) IncrementCodeUse(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE discount_codes SET use_count=use_count+1, updated_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *PostgresRepository) GetCategoriesWithActivePromotions(ctx context.Context, categoryIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(categoryIDs) == 0 {
		return nil, nil
	}

	var hasAll bool
	if err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM promotions
			WHERE is_active = TRUE
			  AND target_type = 'all'
			  AND (starts_at IS NULL OR starts_at <= NOW())
			  AND (ends_at   IS NULL OR ends_at   >  NOW())
		)`).Scan(&hasAll); err != nil {
		return nil, err
	}
	if hasAll {
		return categoryIDs, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT pc.category_id
		FROM promotion_categories pc
		JOIN promotions pr ON pr.id = pc.promotion_id
		WHERE pr.is_active = TRUE
		  AND (pr.starts_at IS NULL OR pr.starts_at <= NOW())
		  AND (pr.ends_at   IS NULL OR pr.ends_at   >  NOW())
		  AND pc.category_id = ANY($1)
	`, categoryIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}
