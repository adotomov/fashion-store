package infrastructure

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/auth/domain"
)

type PostgresIdentityRepository struct {
	db *pgxpool.Pool
}

func NewPostgresIdentityRepository(db *pgxpool.Pool) *PostgresIdentityRepository {
	return &PostgresIdentityRepository{db: db}
}

func (r *PostgresIdentityRepository) FindByProviderSubject(ctx context.Context, provider, subject string) (*domain.Identity, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, provider, provider_subject, email, email_verified
		FROM user_identities WHERE provider = $1 AND provider_subject = $2`,
		provider, subject)

	var id domain.Identity
	err := row.Scan(&id.ID, &id.UserID, &id.Provider, &id.ProviderSubject, &id.Email, &id.EmailVerified)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrIdentityNotFound
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (r *PostgresIdentityRepository) Create(ctx context.Context, identity domain.Identity) (*domain.Identity, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO user_identities (user_id, provider, provider_subject, email, email_verified)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, provider, provider_subject, email, email_verified`,
		identity.UserID, identity.Provider, identity.ProviderSubject, identity.Email, identity.EmailVerified)

	var created domain.Identity
	err := row.Scan(&created.ID, &created.UserID, &created.Provider, &created.ProviderSubject, &created.Email, &created.EmailVerified)
	if err != nil {
		return nil, err
	}
	return &created, nil
}

type PostgresSessionRepository struct {
	db *pgxpool.Pool
}

func NewPostgresSessionRepository(db *pgxpool.Pool) *PostgresSessionRepository {
	return &PostgresSessionRepository{db: db}
}

func (r *PostgresSessionRepository) Create(ctx context.Context, session domain.Session) (*domain.Session, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO user_sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token_hash, expires_at, revoked_at, created_at`,
		session.UserID, session.TokenHash, session.ExpiresAt)

	return scanSession(row)
}

func (r *PostgresSessionRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM user_sessions WHERE token_hash = $1`, tokenHash)

	return scanSession(row)
}

func (r *PostgresSessionRepository) Revoke(ctx context.Context, sessionID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE user_sessions SET revoked_at = NOW() WHERE id = $1`, sessionID)
	return err
}

func scanSession(row pgx.Row) (*domain.Session, error) {
	var s domain.Session
	err := row.Scan(&s.ID, &s.UserID, &s.TokenHash, &s.ExpiresAt, &s.RevokedAt, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}
