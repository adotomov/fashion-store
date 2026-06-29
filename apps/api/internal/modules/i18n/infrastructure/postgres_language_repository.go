package infrastructure

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/i18n/domain"
)

type PostgresLanguageRepository struct {
	db *pgxpool.Pool
}

func NewPostgresLanguageRepository(db *pgxpool.Pool) *PostgresLanguageRepository {
	return &PostgresLanguageRepository{db: db}
}

func (r *PostgresLanguageRepository) List(ctx context.Context) ([]domain.Language, error) {
	rows, err := r.db.Query(ctx, `SELECT code, name, is_default, enabled FROM store_languages ORDER BY is_default DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Language
	for rows.Next() {
		var l domain.Language
		if err := rows.Scan(&l.Code, &l.Name, &l.IsDefault, &l.Enabled); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *PostgresLanguageRepository) Get(ctx context.Context, code string) (*domain.Language, error) {
	row := r.db.QueryRow(ctx, `SELECT code, name, is_default, enabled FROM store_languages WHERE code = $1`, code)
	return scanLanguage(row)
}

func (r *PostgresLanguageRepository) Create(ctx context.Context, lang domain.Language) (*domain.Language, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO store_languages (code, name, is_default, enabled)
		VALUES ($1, $2, $3, $4)
		RETURNING code, name, is_default, enabled`,
		lang.Code, lang.Name, lang.IsDefault, lang.Enabled)
	return scanLanguage(row)
}

func (r *PostgresLanguageRepository) SetEnabled(ctx context.Context, code string, enabled bool) (*domain.Language, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE store_languages SET enabled = $2 WHERE code = $1
		RETURNING code, name, is_default, enabled`, code, enabled)
	return scanLanguage(row)
}

func (r *PostgresLanguageRepository) Delete(ctx context.Context, code string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM store_languages WHERE code = $1`, code)
	return err
}

func scanLanguage(row pgx.Row) (*domain.Language, error) {
	var l domain.Language
	if err := row.Scan(&l.Code, &l.Name, &l.IsDefault, &l.Enabled); err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrLanguageNotFound
		}
		return nil, err
	}
	return &l, nil
}
