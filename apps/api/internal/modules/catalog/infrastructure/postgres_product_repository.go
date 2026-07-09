package infrastructure

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

type PostgresProductRepository struct {
	db *pgxpool.Pool
}

func NewPostgresProductRepository(db *pgxpool.Pool) *PostgresProductRepository {
	return &PostgresProductRepository{db: db}
}

func (r *PostgresProductRepository) Stats(ctx context.Context) (application.CatalogStats, error) {
	var stats application.CatalogStats

	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*),
			COUNT(*) FILTER (WHERE status = 'active'),
			COUNT(*) FILTER (WHERE status = 'draft'),
			COUNT(*) FILTER (WHERE status = 'archived')
		FROM products`,
	).Scan(&stats.TotalProducts, &stats.ActiveProducts, &stats.DraftProducts, &stats.ArchivedProducts); err != nil {
		return stats, err
	}

	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM product_variants`).Scan(&stats.TotalVariants); err != nil {
		return stats, err
	}

	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM categories`).Scan(&stats.TotalCategories); err != nil {
		return stats, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT oi.product_id, p.name, SUM(oi.quantity), COUNT(DISTINCT oi.order_id)
		FROM order_items oi
		JOIN products p ON p.id = oi.product_id
		WHERE oi.product_id IS NOT NULL
		GROUP BY oi.product_id, p.name
		ORDER BY SUM(oi.quantity) DESC
		LIMIT 10`)
	if err != nil {
		return stats, err
	}
	defer rows.Close()
	for rows.Next() {
		var tp application.TopProduct
		if err := rows.Scan(&tp.ProductID, &tp.ProductName, &tp.QuantitySold, &tp.OrderCount); err != nil {
			return stats, err
		}
		stats.TopProducts = append(stats.TopProducts, tp)
	}
	if err := rows.Err(); err != nil {
		return stats, err
	}

	return stats, nil
}

func (r *PostgresProductRepository) Create(ctx context.Context, product domain.Product) (*domain.Product, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO products (name, slug, description, status, base_price_amount, base_price_currency)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, slug, COALESCE(description, ''), status, base_price_amount, base_price_currency,
			compare_at_price_amount, compare_at_price_currency, created_at, updated_at, tax_group_id`,
		product.Name, product.Slug, product.Description, product.Status,
		product.BasePrice.AmountMinor, product.BasePrice.Currency)

	created, err := scanProduct(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, application.ErrSlugConflict
		}
		return nil, err
	}
	return created, nil
}

func (r *PostgresProductRepository) List(ctx context.Context) ([]domain.Product, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.id, p.name, p.slug, COALESCE(p.description, ''), p.status, p.base_price_amount, p.base_price_currency,
		       p.compare_at_price_amount, p.compare_at_price_currency, p.created_at, p.updated_at, COUNT(v.id),
		       (SELECT m.id FROM product_media m WHERE m.product_id = p.id ORDER BY m.position LIMIT 1),
		       (SELECT m.bucket FROM product_media m WHERE m.product_id = p.id ORDER BY m.position LIMIT 1),
		       (SELECT m.object_key FROM product_media m WHERE m.product_id = p.id ORDER BY m.position LIMIT 1),
		       (SELECT m.content_type FROM product_media m WHERE m.product_id = p.id ORDER BY m.position LIMIT 1),
		       (NOT EXISTS (SELECT 1 FROM product_variants pv WHERE pv.product_id = p.id)
		        OR EXISTS (
		            SELECT 1 FROM product_variants pv
		            JOIN inventory_items ii ON ii.variant_id = pv.id
		            WHERE pv.product_id = p.id AND (ii.quantity_on_hand - ii.quantity_reserved) > 0
		        ))
		FROM products p
		LEFT JOIN product_variants v ON v.product_id = p.id
		GROUP BY p.id
		ORDER BY p.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var p domain.Product
		var amount int64
		var currency string
		var compareAmount *int64
		var compareCurrency *string
		var mediaID *uuid.UUID
		var mediaBucket, mediaObjectKey, mediaContentType *string
		if err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.Status, &amount, &currency,
			&compareAmount, &compareCurrency, &p.CreatedAt, &p.UpdatedAt, &p.VariantCount,
			&mediaID, &mediaBucket, &mediaObjectKey, &mediaContentType, &p.InStock); err != nil {
			return nil, err
		}
		p.BasePrice = money.Money{AmountMinor: amount, Currency: currency}
		if compareAmount != nil && compareCurrency != nil {
			p.CompareAtPrice = &money.Money{AmountMinor: *compareAmount, Currency: *compareCurrency}
		}
		if mediaID != nil {
			p.PrimaryMedia = &domain.ProductMedia{
				ID:          *mediaID,
				ProductID:   p.ID,
				Bucket:      *mediaBucket,
				ObjectKey:   *mediaObjectKey,
				ContentType: *mediaContentType,
			}
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (r *PostgresProductRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, name, slug, COALESCE(description, ''), status, base_price_amount, base_price_currency,
			compare_at_price_amount, compare_at_price_currency, created_at, updated_at, tax_group_id
		FROM products WHERE id = $1`, id)

	product, err := scanProduct(row)
	if err != nil {
		return nil, err
	}
	return r.loadProductRelations(ctx, product)
}

func (r *PostgresProductRepository) FindBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, name, slug, COALESCE(description, ''), status, base_price_amount, base_price_currency,
			compare_at_price_amount, compare_at_price_currency, created_at, updated_at, tax_group_id
		FROM products WHERE slug = $1`, slug)

	product, err := scanProduct(row)
	if err != nil {
		return nil, err
	}
	return r.loadProductRelations(ctx, product)
}

// loadProductRelations populates the sub-resources only FindByID/FindBySlug
// need — List() stays lightweight and skips this entirely.
func (r *PostgresProductRepository) loadProductRelations(ctx context.Context, product *domain.Product) (*domain.Product, error) {
	var err error
	if product.CategoryIDs, err = r.categoryIDsFor(ctx, product.ID); err != nil {
		return nil, err
	}
	if product.CatalogIDs, err = r.catalogIDsFor(ctx, product.ID); err != nil {
		return nil, err
	}
	if product.Attributes, err = r.attributeRefsFor(ctx, product.ID); err != nil {
		return nil, err
	}
	if product.Media, err = r.mediaFor(ctx, product.ID); err != nil {
		return nil, err
	}
	if product.Variants, err = r.variantsFor(ctx, product.ID); err != nil {
		return nil, err
	}
	product.VariantCount = len(product.Variants)
	if len(product.Media) > 0 {
		product.PrimaryMedia = &product.Media[0]
	}

	product.InStock = len(product.Variants) == 0
	for _, v := range product.Variants {
		if v.QuantityAvailable != nil && *v.QuantityAvailable > 0 {
			product.InStock = true
			break
		}
	}

	return product, nil
}

func (r *PostgresProductRepository) Update(ctx context.Context, product domain.Product) (*domain.Product, error) {
	var compareAmount *int64
	var compareCurrency *string
	if product.CompareAtPrice != nil {
		compareAmount = &product.CompareAtPrice.AmountMinor
		compareCurrency = &product.CompareAtPrice.Currency
	}

	row := r.db.QueryRow(ctx, `
		UPDATE products SET name = $2, description = $3, status = $4,
			base_price_amount = $5, base_price_currency = $6,
			compare_at_price_amount = $7, compare_at_price_currency = $8,
			tax_group_id = $9, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, slug, COALESCE(description, ''), status, base_price_amount, base_price_currency,
			compare_at_price_amount, compare_at_price_currency, created_at, updated_at, tax_group_id`,
		product.ID, product.Name, product.Description, product.Status,
		product.BasePrice.AmountMinor, product.BasePrice.Currency, compareAmount, compareCurrency,
		product.TaxGroupID)

	return scanProduct(row)
}

func (r *PostgresProductRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}

func (r *PostgresProductRepository) SetCategories(ctx context.Context, productID uuid.UUID, categoryIDs []uuid.UUID) error {
	return r.replaceJoinRows(ctx, "product_categories", "product_id", "category_id", productID, categoryIDs)
}

func (r *PostgresProductRepository) SetCatalogs(ctx context.Context, productID uuid.UUID, catalogIDs []uuid.UUID) error {
	return r.replaceJoinRows(ctx, "catalog_products", "product_id", "catalog_id", productID, catalogIDs)
}

func (r *PostgresProductRepository) SetAttributes(ctx context.Context, productID uuid.UUID, attributeIDs []uuid.UUID) error {
	return r.replaceJoinRows(ctx, "product_attributes", "product_id", "attribute_id", productID, attributeIDs)
}

func (r *PostgresProductRepository) attributeRefsFor(ctx context.Context, productID uuid.UUID) ([]domain.AttributeRef, error) {
	rows, err := r.db.Query(ctx, `
		SELECT a.id, a.name, a.type
		FROM product_attributes pa
		JOIN attributes a ON a.id = pa.attribute_id
		WHERE pa.product_id = $1
		ORDER BY a.name`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	refs := []domain.AttributeRef{}
	for rows.Next() {
		var ref domain.AttributeRef
		if err := rows.Scan(&ref.ID, &ref.Name, &ref.Type); err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, rows.Err()
}

// replaceJoinRows performs a delete-then-insert replace of a many-to-many
// join table's rows for a single owning ID, inside one transaction.
func (r *PostgresProductRepository) replaceJoinRows(ctx context.Context, table, ownerColumn, otherColumn string, ownerID uuid.UUID, otherIDs []uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM `+table+` WHERE `+ownerColumn+` = $1`, ownerID); err != nil {
		return err
	}

	for _, otherID := range otherIDs {
		query := `INSERT INTO ` + table + ` (` + ownerColumn + `, ` + otherColumn + `) VALUES ($1, $2)`
		if _, err := tx.Exec(ctx, query, ownerID, otherID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresProductRepository) categoryIDsFor(ctx context.Context, productID uuid.UUID) ([]uuid.UUID, error) {
	return r.idsFor(ctx, "product_categories", "product_id", "category_id", productID)
}

func (r *PostgresProductRepository) catalogIDsFor(ctx context.Context, productID uuid.UUID) ([]uuid.UUID, error) {
	return r.idsFor(ctx, "catalog_products", "product_id", "catalog_id", productID)
}

func (r *PostgresProductRepository) ProductIDsByCategory(ctx context.Context, categoryID uuid.UUID) ([]uuid.UUID, error) {
	return r.idsFor(ctx, "product_categories", "category_id", "product_id", categoryID)
}

func (r *PostgresProductRepository) ProductIDsByCatalog(ctx context.Context, catalogID uuid.UUID) ([]uuid.UUID, error) {
	return r.idsFor(ctx, "catalog_products", "catalog_id", "product_id", catalogID)
}

func (r *PostgresProductRepository) BestInCategoryProductIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT ON (pc.category_id) p.id
		FROM products p
		JOIN product_categories pc ON pc.product_id = p.id
		WHERE p.status = 'active'
		ORDER BY pc.category_id, p.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *PostgresProductRepository) ProductIDsByAttributeValues(ctx context.Context, valueIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(valueIDs) == 0 {
		return []uuid.UUID{}, nil
	}

	var groupCount int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT attribute_id) FROM attribute_values WHERE id = ANY($1)`,
		valueIDs).Scan(&groupCount); err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT pv.product_id, av.attribute_id
		FROM product_variants pv
		JOIN variant_attribute_values vav ON vav.variant_id = pv.id
		JOIN attribute_values av ON av.id = vav.attribute_value_id
		WHERE vav.attribute_value_id = ANY($1)`,
		valueIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matchedGroups := map[uuid.UUID]map[uuid.UUID]bool{}
	for rows.Next() {
		var productID, attributeID uuid.UUID
		if err := rows.Scan(&productID, &attributeID); err != nil {
			return nil, err
		}
		if matchedGroups[productID] == nil {
			matchedGroups[productID] = map[uuid.UUID]bool{}
		}
		matchedGroups[productID][attributeID] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	ids := []uuid.UUID{}
	for productID, groups := range matchedGroups {
		if len(groups) == groupCount {
			ids = append(ids, productID)
		}
	}
	return ids, nil
}

func (r *PostgresProductRepository) AttributeFacets(ctx context.Context, categoryIDs []uuid.UUID, catalogID *uuid.UUID) ([]domain.AttributeFacet, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT a.id, a.name, a.type, av.id, av.value, av.color_hex
		FROM variant_attribute_values vav
		JOIN attribute_values av ON av.id = vav.attribute_value_id
		JOIN attributes a ON a.id = av.attribute_id
		JOIN product_variants pv ON pv.id = vav.variant_id
		JOIN products p ON p.id = pv.product_id
		WHERE p.status = 'active'
			AND ($1::uuid[] IS NULL OR p.id IN (SELECT product_id FROM product_categories WHERE category_id = ANY($1)))
			AND ($2::uuid IS NULL OR p.id IN (SELECT product_id FROM catalog_products WHERE catalog_id = $2))
		ORDER BY a.name, av.value`,
		nullableUUIDSlice(categoryIDs), catalogID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	order := []uuid.UUID{}
	byID := map[uuid.UUID]*domain.AttributeFacet{}
	for rows.Next() {
		var attrID, valueID uuid.UUID
		var attrName, value string
		var attrType domain.AttributeType
		var colorHex *string
		if err := rows.Scan(&attrID, &attrName, &attrType, &valueID, &value, &colorHex); err != nil {
			return nil, err
		}
		facet, ok := byID[attrID]
		if !ok {
			facet = &domain.AttributeFacet{AttributeID: attrID, AttributeName: attrName, AttributeType: attrType}
			byID[attrID] = facet
			order = append(order, attrID)
		}
		facet.Values = append(facet.Values, domain.AttributeFacetValue{ID: valueID, Value: value, ColorHex: colorHex})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	facets := make([]domain.AttributeFacet, 0, len(order))
	for _, id := range order {
		facets = append(facets, *byID[id])
	}
	return facets, nil
}

// nullableUUIDSlice lets an empty/nil slice bind as SQL NULL instead of an
// empty array, since `= ANY('{}')` is always false but `IS NULL` lets the
// surrounding OR correctly mean "no category filter requested".
func nullableUUIDSlice(ids []uuid.UUID) interface{} {
	if len(ids) == 0 {
		return nil
	}
	return ids
}

func (r *PostgresProductRepository) idsFor(ctx context.Context, table, ownerColumn, otherColumn string, ownerID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `SELECT `+otherColumn+` FROM `+table+` WHERE `+ownerColumn+` = $1`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// attributeValuesFor loads attribute values joined through the given join
// table (variant_attribute_values, via variantsFor).
func (r *PostgresProductRepository) attributeValuesFor(ctx context.Context, table, ownerColumn string, ownerID uuid.UUID) ([]domain.AttributeValue, error) {
	rows, err := r.db.Query(ctx, `
		SELECT av.id, av.attribute_id, av.value, av.color_hex, av.created_at
		FROM `+table+` j
		JOIN attribute_values av ON av.id = j.attribute_value_id
		WHERE j.`+ownerColumn+` = $1
		ORDER BY av.value`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := []domain.AttributeValue{}
	for rows.Next() {
		var v domain.AttributeValue
		if err := rows.Scan(&v.ID, &v.AttributeID, &v.Value, &v.ColorHex, &v.CreatedAt); err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, rows.Err()
}

func scanProduct(row pgx.Row) (*domain.Product, error) {
	var p domain.Product
	var amount int64
	var currency string
	var compareAmount *int64
	var compareCurrency *string
	err := row.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.Status, &amount, &currency,
		&compareAmount, &compareCurrency, &p.CreatedAt, &p.UpdatedAt, &p.TaxGroupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrProductNotFound
	}
	if err != nil {
		return nil, err
	}
	p.BasePrice = money.Money{AmountMinor: amount, Currency: currency}
	if compareAmount != nil && compareCurrency != nil {
		p.CompareAtPrice = &money.Money{AmountMinor: *compareAmount, Currency: *compareCurrency}
	}
	return &p, nil
}

func (r *PostgresProductRepository) GetTaxGroupID(ctx context.Context, productID uuid.UUID) (*uuid.UUID, error) {
	var id *uuid.UUID
	err := r.db.QueryRow(ctx, `SELECT tax_group_id FROM products WHERE id = $1`, productID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return id, err
}
