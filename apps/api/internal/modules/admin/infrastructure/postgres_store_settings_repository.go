package infrastructure

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/admin/domain"
)

type PostgresStoreSettingsRepository struct {
	db *pgxpool.Pool
}

func NewPostgresStoreSettingsRepository(db *pgxpool.Pool) *PostgresStoreSettingsRepository {
	return &PostgresStoreSettingsRepository{db: db}
}

const storeSettingsColumns = `id, store_name, legal_entity_name, locale, currency,
	contact_email, contact_phone, company_description, facebook_url, instagram_url,
	logo_bucket, logo_object_key, logo_content_type, logo_size_bytes,
	created_at, updated_at`

// Get returns the single store_settings row, seeded by migration — there is
// always exactly one, so ORDER BY + LIMIT 1 avoids depending on a fixed ID.
func (r *PostgresStoreSettingsRepository) Get(ctx context.Context) (*domain.StoreSettings, error) {
	row := r.db.QueryRow(ctx, `SELECT `+storeSettingsColumns+` FROM store_settings ORDER BY created_at LIMIT 1`)
	return scanStoreSettings(row)
}

func (r *PostgresStoreSettingsRepository) Update(ctx context.Context, settings domain.StoreSettings) (*domain.StoreSettings, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE store_settings SET
			store_name = $2, legal_entity_name = $3, locale = $4, currency = $5,
			contact_email = $6, contact_phone = $7, company_description = $8,
			facebook_url = $9, instagram_url = $10,
			logo_bucket = $11, logo_object_key = $12, logo_content_type = $13, logo_size_bytes = $14,
			updated_at = NOW()
		WHERE id = $1
		RETURNING `+storeSettingsColumns,
		settings.ID, settings.StoreName, settings.LegalEntityName, settings.Locale, settings.Currency,
		settings.ContactEmail, settings.ContactPhone, settings.CompanyDescription,
		settings.FacebookURL, settings.InstagramURL,
		settings.LogoBucket, settings.LogoObjectKey, settings.LogoContentType, settings.LogoSizeBytes)

	return scanStoreSettings(row)
}

func scanStoreSettings(row pgx.Row) (*domain.StoreSettings, error) {
	var s domain.StoreSettings
	err := row.Scan(
		&s.ID, &s.StoreName, &s.LegalEntityName, &s.Locale, &s.Currency,
		&s.ContactEmail, &s.ContactPhone, &s.CompanyDescription, &s.FacebookURL, &s.InstagramURL,
		&s.LogoBucket, &s.LogoObjectKey, &s.LogoContentType, &s.LogoSizeBytes,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PostgresStoreSettingsRepository) GetHeroSettings(ctx context.Context) (domain.HeroSettings, error) {
	var s domain.HeroSettings
	err := r.db.QueryRow(ctx, `
		SELECT eyebrow, heading, subtext,
		       cta_primary_label, cta_primary_url,
		       cta_secondary_label, cta_secondary_url,
		       background_image_bucket, background_image_object_key,
		       background_image_content_type, background_image_size_bytes,
		       updated_at
		FROM hero_settings LIMIT 1
	`).Scan(
		&s.Eyebrow, &s.Heading, &s.Subtext,
		&s.CTAPrimaryLabel, &s.CTAPrimaryURL,
		&s.CTASecondaryLabel, &s.CTASecondaryURL,
		&s.BackgroundBucket, &s.BackgroundObjectKey,
		&s.BackgroundContentType, &s.BackgroundSizeBytes,
		&s.UpdatedAt,
	)
	return s, err
}

func (r *PostgresStoreSettingsRepository) SaveHeroSettings(ctx context.Context, s domain.HeroSettings) (domain.HeroSettings, error) {
	err := r.db.QueryRow(ctx, `
		INSERT INTO hero_settings (id, eyebrow, heading, subtext,
		    cta_primary_label, cta_primary_url,
		    cta_secondary_label, cta_secondary_url,
		    background_image_bucket, background_image_object_key,
		    background_image_content_type, background_image_size_bytes,
		    updated_at)
		VALUES (TRUE, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		ON CONFLICT (id) DO UPDATE SET
		    eyebrow                       = EXCLUDED.eyebrow,
		    heading                       = EXCLUDED.heading,
		    subtext                       = EXCLUDED.subtext,
		    cta_primary_label             = EXCLUDED.cta_primary_label,
		    cta_primary_url               = EXCLUDED.cta_primary_url,
		    cta_secondary_label           = EXCLUDED.cta_secondary_label,
		    cta_secondary_url             = EXCLUDED.cta_secondary_url,
		    background_image_bucket       = EXCLUDED.background_image_bucket,
		    background_image_object_key   = EXCLUDED.background_image_object_key,
		    background_image_content_type = EXCLUDED.background_image_content_type,
		    background_image_size_bytes   = EXCLUDED.background_image_size_bytes,
		    updated_at                    = NOW()
		RETURNING eyebrow, heading, subtext,
		    cta_primary_label, cta_primary_url,
		    cta_secondary_label, cta_secondary_url,
		    background_image_bucket, background_image_object_key,
		    background_image_content_type, background_image_size_bytes,
		    updated_at
	`,
		s.Eyebrow, s.Heading, s.Subtext,
		s.CTAPrimaryLabel, s.CTAPrimaryURL,
		s.CTASecondaryLabel, s.CTASecondaryURL,
		s.BackgroundBucket, s.BackgroundObjectKey,
		s.BackgroundContentType, s.BackgroundSizeBytes,
	).Scan(
		&s.Eyebrow, &s.Heading, &s.Subtext,
		&s.CTAPrimaryLabel, &s.CTAPrimaryURL,
		&s.CTASecondaryLabel, &s.CTASecondaryURL,
		&s.BackgroundBucket, &s.BackgroundObjectKey,
		&s.BackgroundContentType, &s.BackgroundSizeBytes,
		&s.UpdatedAt,
	)
	return s, err
}

func (r *PostgresStoreSettingsRepository) GetEditorialBanner(ctx context.Context) (domain.EditorialBanner, error) {
	var b domain.EditorialBanner
	err := r.db.QueryRow(ctx, `
		SELECT enabled, eyebrow, heading, subtext, cta_label, cta_url,
		       image_bucket, image_object_key, image_content_type, image_size_bytes,
		       updated_at
		FROM editorial_banner_settings LIMIT 1
	`).Scan(
		&b.Enabled, &b.Eyebrow, &b.Heading, &b.Subtext, &b.CTALabel, &b.CTAURL,
		&b.ImageBucket, &b.ImageObjectKey, &b.ImageContentType, &b.ImageSizeBytes,
		&b.UpdatedAt,
	)
	return b, err
}

func (r *PostgresStoreSettingsRepository) SaveEditorialBanner(ctx context.Context, b domain.EditorialBanner) (domain.EditorialBanner, error) {
	err := r.db.QueryRow(ctx, `
		INSERT INTO editorial_banner_settings (id, enabled, eyebrow, heading, subtext,
		    cta_label, cta_url,
		    image_bucket, image_object_key, image_content_type, image_size_bytes,
		    updated_at)
		VALUES (TRUE, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		ON CONFLICT (id) DO UPDATE SET
		    enabled            = EXCLUDED.enabled,
		    eyebrow            = EXCLUDED.eyebrow,
		    heading            = EXCLUDED.heading,
		    subtext            = EXCLUDED.subtext,
		    cta_label          = EXCLUDED.cta_label,
		    cta_url            = EXCLUDED.cta_url,
		    image_bucket       = EXCLUDED.image_bucket,
		    image_object_key   = EXCLUDED.image_object_key,
		    image_content_type = EXCLUDED.image_content_type,
		    image_size_bytes   = EXCLUDED.image_size_bytes,
		    updated_at         = NOW()
		RETURNING enabled, eyebrow, heading, subtext, cta_label, cta_url,
		    image_bucket, image_object_key, image_content_type, image_size_bytes,
		    updated_at
	`,
		b.Enabled, b.Eyebrow, b.Heading, b.Subtext, b.CTALabel, b.CTAURL,
		b.ImageBucket, b.ImageObjectKey, b.ImageContentType, b.ImageSizeBytes,
	).Scan(
		&b.Enabled, &b.Eyebrow, &b.Heading, &b.Subtext, &b.CTALabel, &b.CTAURL,
		&b.ImageBucket, &b.ImageObjectKey, &b.ImageContentType, &b.ImageSizeBytes,
		&b.UpdatedAt,
	)
	return b, err
}

func (r *PostgresStoreSettingsRepository) ListHomeSections(ctx context.Context) ([]domain.HomeSection, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, enabled, eyebrow, heading, updated_at
		FROM home_sections ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sections []domain.HomeSection
	for rows.Next() {
		var s domain.HomeSection
		if err := rows.Scan(&s.ID, &s.Enabled, &s.Eyebrow, &s.Heading, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sections = append(sections, s)
	}
	return sections, rows.Err()
}

func (r *PostgresStoreSettingsRepository) SaveHomeSection(ctx context.Context, s domain.HomeSection) (domain.HomeSection, error) {
	var result domain.HomeSection
	err := r.db.QueryRow(ctx, `
		UPDATE home_sections
		SET enabled = $2, eyebrow = $3, heading = $4, updated_at = NOW()
		WHERE id = $1
		RETURNING id, enabled, eyebrow, heading, updated_at
	`, s.ID, s.Enabled, s.Eyebrow, s.Heading).Scan(
		&result.ID, &result.Enabled, &result.Eyebrow, &result.Heading, &result.UpdatedAt,
	)
	return result, err
}

func (r *PostgresStoreSettingsRepository) GetSectionProductIDs(ctx context.Context, sectionID string) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `
		SELECT product_id FROM home_section_products
		WHERE section_id = $1
		ORDER BY sort_order, product_id
	`, sectionID)
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

func (r *PostgresStoreSettingsRepository) SetSectionProducts(ctx context.Context, sectionID string, productIDs []uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM home_section_products WHERE section_id = $1`, sectionID); err != nil {
		return err
	}
	for i, productID := range productIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO home_section_products (section_id, product_id, sort_order)
			VALUES ($1, $2, $3)
		`, sectionID, productID, i); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PostgresStoreSettingsRepository) GetSectionCategoryGroups(ctx context.Context, sectionID string) ([]domain.SectionCategoryGroup, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.category_id, p.product_id
		FROM home_section_categories c
		LEFT JOIN home_section_category_products p
		       ON p.section_id = c.section_id AND p.category_id = c.category_id
		WHERE c.section_id = $1
		ORDER BY c.sort_order, c.category_id, p.sort_order, p.product_id
	`, sectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Preserve the category order from the query while grouping products.
	var groups []domain.SectionCategoryGroup
	index := map[uuid.UUID]int{}
	for rows.Next() {
		var categoryID uuid.UUID
		var productID *uuid.UUID
		if err := rows.Scan(&categoryID, &productID); err != nil {
			return nil, err
		}
		i, ok := index[categoryID]
		if !ok {
			i = len(groups)
			index[categoryID] = i
			groups = append(groups, domain.SectionCategoryGroup{CategoryID: categoryID})
		}
		if productID != nil {
			groups[i].ProductIDs = append(groups[i].ProductIDs, *productID)
		}
	}
	return groups, rows.Err()
}

func (r *PostgresStoreSettingsRepository) SetSectionCategoryGroups(ctx context.Context, sectionID string, groups []domain.SectionCategoryGroup) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Deleting the categories cascades to home_section_category_products.
	if _, err := tx.Exec(ctx, `DELETE FROM home_section_categories WHERE section_id = $1`, sectionID); err != nil {
		return err
	}
	for ci, group := range groups {
		if _, err := tx.Exec(ctx, `
			INSERT INTO home_section_categories (section_id, category_id, sort_order)
			VALUES ($1, $2, $3)
		`, sectionID, group.CategoryID, ci); err != nil {
			return err
		}
		for pi, productID := range group.ProductIDs {
			if _, err := tx.Exec(ctx, `
				INSERT INTO home_section_category_products (section_id, category_id, product_id, sort_order)
				VALUES ($1, $2, $3, $4)
			`, sectionID, group.CategoryID, productID, pi); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}
