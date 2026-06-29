package infrastructure

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/admin/domain"
)

type PostgresStoreAddressRepository struct {
	db *pgxpool.Pool
}

func NewPostgresStoreAddressRepository(db *pgxpool.Pool) *PostgresStoreAddressRepository {
	return &PostgresStoreAddressRepository{db: db}
}

const storeAddressColumns = `id, store_settings_id, label, line1, line2, city, region, postal_code, country, is_default, created_at, updated_at`

func (r *PostgresStoreAddressRepository) List(ctx context.Context, storeSettingsID uuid.UUID) ([]domain.StoreAddress, error) {
	rows, err := r.db.Query(ctx, `SELECT `+storeAddressColumns+` FROM store_addresses WHERE store_settings_id = $1 ORDER BY is_default DESC, created_at`, storeSettingsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.StoreAddress
	for rows.Next() {
		a, err := scanStoreAddress(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *PostgresStoreAddressRepository) Get(ctx context.Context, id uuid.UUID) (*domain.StoreAddress, error) {
	row := r.db.QueryRow(ctx, `SELECT `+storeAddressColumns+` FROM store_addresses WHERE id = $1`, id)
	return scanStoreAddress(row)
}

func (r *PostgresStoreAddressRepository) Create(ctx context.Context, a domain.StoreAddress) (*domain.StoreAddress, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO store_addresses (store_settings_id, label, line1, line2, city, region, postal_code, country, is_default)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING `+storeAddressColumns,
		a.StoreSettingsID, a.Label, a.Line1, a.Line2, a.City, a.Region, a.PostalCode, a.Country, a.IsDefault)
	return scanStoreAddress(row)
}

func (r *PostgresStoreAddressRepository) Update(ctx context.Context, a domain.StoreAddress) (*domain.StoreAddress, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE store_addresses SET
			label = $2, line1 = $3, line2 = $4, city = $5, region = $6, postal_code = $7, country = $8, is_default = $9,
			updated_at = NOW()
		WHERE id = $1
		RETURNING `+storeAddressColumns,
		a.ID, a.Label, a.Line1, a.Line2, a.City, a.Region, a.PostalCode, a.Country, a.IsDefault)
	return scanStoreAddress(row)
}

func (r *PostgresStoreAddressRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM store_addresses WHERE id = $1`, id)
	return err
}

func (r *PostgresStoreAddressRepository) ClearDefault(ctx context.Context, storeSettingsID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE store_addresses SET is_default = false WHERE store_settings_id = $1`, storeSettingsID)
	return err
}

func scanStoreAddress(row pgx.Row) (*domain.StoreAddress, error) {
	var a domain.StoreAddress
	err := row.Scan(&a.ID, &a.StoreSettingsID, &a.Label, &a.Line1, &a.Line2, &a.City, &a.Region, &a.PostalCode, &a.Country, &a.IsDefault, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrAddressNotFound
		}
		return nil, err
	}
	return &a, nil
}
