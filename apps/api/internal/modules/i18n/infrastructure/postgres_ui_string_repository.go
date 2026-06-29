package infrastructure

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/i18n/domain"
)

type PostgresUIStringRepository struct {
	db *pgxpool.Pool
}

func NewPostgresUIStringRepository(db *pgxpool.Pool) *PostgresUIStringRepository {
	return &PostgresUIStringRepository{db: db}
}

func (r *PostgresUIStringRepository) ListAll(ctx context.Context) ([]domain.UIString, error) {
	rows, err := r.db.Query(ctx, `SELECT key, locale, value FROM ui_strings ORDER BY key, locale`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.UIString
	for rows.Next() {
		var s domain.UIString
		if err := rows.Scan(&s.Key, &s.Locale, &s.Value); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PostgresUIStringRepository) GetByLocale(ctx context.Context, locale string) (map[string]string, error) {
	rows, err := r.db.Query(ctx, `SELECT key, value FROM ui_strings WHERE locale = $1`, locale)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]string{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		out[key] = value
	}
	return out, rows.Err()
}

func (r *PostgresUIStringRepository) Upsert(ctx context.Context, s domain.UIString) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO ui_strings (key, locale, value)
		VALUES ($1, $2, $3)
		ON CONFLICT (key, locale) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
		s.Key, s.Locale, s.Value)
	return err
}

func (r *PostgresUIStringRepository) SeedDefaults(ctx context.Context, defaults map[string]string) error {
	batch := &pgx.Batch{}
	for key, value := range defaults {
		batch.Queue(`
			INSERT INTO ui_strings (key, locale, value) VALUES ($1, 'en', $2)
			ON CONFLICT (key, locale) DO NOTHING`, key, value)
	}
	br := r.db.SendBatch(ctx, batch)
	defer br.Close()
	for range defaults {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}
