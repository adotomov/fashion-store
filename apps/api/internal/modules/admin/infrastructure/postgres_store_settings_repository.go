package infrastructure

import (
	"context"

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
	contact_email, contact_phone, company_description,
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
			logo_bucket = $9, logo_object_key = $10, logo_content_type = $11, logo_size_bytes = $12,
			updated_at = NOW()
		WHERE id = $1
		RETURNING `+storeSettingsColumns,
		settings.ID, settings.StoreName, settings.LegalEntityName, settings.Locale, settings.Currency,
		settings.ContactEmail, settings.ContactPhone, settings.CompanyDescription,
		settings.LogoBucket, settings.LogoObjectKey, settings.LogoContentType, settings.LogoSizeBytes)

	return scanStoreSettings(row)
}

func scanStoreSettings(row pgx.Row) (*domain.StoreSettings, error) {
	var s domain.StoreSettings
	err := row.Scan(
		&s.ID, &s.StoreName, &s.LegalEntityName, &s.Locale, &s.Currency,
		&s.ContactEmail, &s.ContactPhone, &s.CompanyDescription,
		&s.LogoBucket, &s.LogoObjectKey, &s.LogoContentType, &s.LogoSizeBytes,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
