package infrastructure

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresSuppressionStore reads and writes the do-not-mail list. Addresses are
// normalised to lowercase so a differently-cased bounce still suppresses future
// sends to the same mailbox.
type PostgresSuppressionStore struct {
	db *pgxpool.Pool
}

func NewPostgresSuppressionStore(db *pgxpool.Pool) *PostgresSuppressionStore {
	return &PostgresSuppressionStore{db: db}
}

func (s *PostgresSuppressionStore) IsSuppressed(ctx context.Context, email string) (bool, error) {
	const query = `SELECT EXISTS (SELECT 1 FROM email_suppressions WHERE email = $1)`
	var exists bool
	if err := s.db.QueryRow(ctx, query, normaliseEmail(email)).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

// Suppress records an address as undeliverable. Idempotent: a repeated bounce
// keeps the original reason rather than churning the row.
func (s *PostgresSuppressionStore) Suppress(ctx context.Context, email, reason, detail string) error {
	const query = `
		INSERT INTO email_suppressions (email, reason, detail)
		VALUES ($1, $2, NULLIF($3, ''))
		ON CONFLICT (email) DO NOTHING`
	_, err := s.db.Exec(ctx, query, normaliseEmail(email), reason, detail)
	return err
}

func normaliseEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
