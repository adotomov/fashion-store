package infrastructure

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/fulfillment/domain"
)

type PostgresSettingsRepository struct {
	db *pgxpool.Pool
}

func NewPostgresSettingsRepository(db *pgxpool.Pool) *PostgresSettingsRepository {
	return &PostgresSettingsRepository{db: db}
}

func (r *PostgresSettingsRepository) Get(ctx context.Context, provider string) (*domain.ProviderSettings, error) {
	row := r.db.QueryRow(ctx, `
		SELECT provider, enabled, config, updated_at
		FROM logistics_provider_settings WHERE provider = $1`, provider)

	settings, err := scanProviderSettings(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return settings, nil
}

func (r *PostgresSettingsRepository) Save(ctx context.Context, settings domain.ProviderSettings) (*domain.ProviderSettings, error) {
	configJSON, err := json.Marshal(settings.Config)
	if err != nil {
		return nil, err
	}

	row := r.db.QueryRow(ctx, `
		INSERT INTO logistics_provider_settings (provider, enabled, config, updated_at)
		VALUES ($1, $2, $3::jsonb, NOW())
		ON CONFLICT (provider) DO UPDATE SET enabled = $2, config = $3::jsonb, updated_at = NOW()
		RETURNING provider, enabled, config, updated_at`,
		settings.Provider, settings.Enabled, configJSON)

	return scanProviderSettings(row)
}

func (r *PostgresSettingsRepository) List(ctx context.Context) ([]domain.ProviderSettings, error) {
	rows, err := r.db.Query(ctx, `SELECT provider, enabled, config, updated_at FROM logistics_provider_settings ORDER BY provider`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ProviderSettings
	for rows.Next() {
		settings, err := scanProviderSettings(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *settings)
	}
	return result, rows.Err()
}

func scanProviderSettings(row pgx.Row) (*domain.ProviderSettings, error) {
	var s domain.ProviderSettings
	var configJSON []byte
	if err := row.Scan(&s.Provider, &s.Enabled, &configJSON, &s.UpdatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(configJSON, &s.Config); err != nil {
		return nil, err
	}
	return &s, nil
}
