package infrastructure

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/notifications/domain"
)

type PostgresTemplateStore struct {
	db *pgxpool.Pool
}

func NewPostgresTemplateStore(db *pgxpool.Pool) *PostgresTemplateStore {
	return &PostgresTemplateStore{db: db}
}

func (s *PostgresTemplateStore) Get(ctx context.Context, key, locale string) (domain.Template, error) {
	const query = `
		SELECT template_key, locale, subject, html_body, text_body, updated_at
		FROM email_templates
		WHERE template_key = $1 AND locale = $2`

	var tmpl domain.Template
	err := s.db.QueryRow(ctx, query, key, locale).Scan(
		&tmpl.Key, &tmpl.Locale, &tmpl.Subject, &tmpl.HTMLBody, &tmpl.TextBody, &tmpl.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		// Signals "try the default locale" to the service's fallback path.
		return domain.Template{}, domain.ErrTemplateNotFound
	}
	if err != nil {
		return domain.Template{}, err
	}
	return tmpl, nil
}
