package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/notifications/domain"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Enqueue relies on the unique index on dedupe_key for idempotency: ON CONFLICT
// DO NOTHING means a duplicate producer call is a silent no-op, reported to the
// caller as inserted=false rather than an error.
func (r *PostgresRepository) Enqueue(ctx context.Context, msg domain.Message) (bool, error) {
	payload, err := json.Marshal(msg.Payload)
	if err != nil {
		return false, fmt.Errorf("marshal email payload: %w", err)
	}

	var category *string
	if msg.Category != "" {
		category = &msg.Category
	}

	const query = `
		INSERT INTO email_messages (template_key, locale, to_email, to_name, payload, dedupe_key, category)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (dedupe_key) DO NOTHING
		RETURNING id`

	var id uuid.UUID
	err = r.db.QueryRow(ctx, query,
		msg.TemplateKey, msg.Locale, msg.ToEmail, msg.ToName, payload, msg.DedupeKey, category,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil // already queued for this event
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ClaimDue leases a batch of due messages in a single statement. SKIP LOCKED
// lets several dispatchers (or a rolling deploy's overlapping instances) run
// concurrently without handing the same message to two senders, and pushing
// next_attempt_at forward means a dispatcher that dies mid-send simply releases
// the row when the lease lapses.
func (r *PostgresRepository) ClaimDue(ctx context.Context, limit int, lease time.Duration) ([]domain.Message, error) {
	const query = `
		UPDATE email_messages SET
			status = 'sending',
			attempts = attempts + 1,
			next_attempt_at = NOW() + $2::interval,
			updated_at = NOW()
		WHERE id IN (
			SELECT id FROM email_messages
			WHERE status IN ('pending', 'sending') AND next_attempt_at <= NOW()
			ORDER BY next_attempt_at
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, template_key, locale, to_email, to_name, payload, dedupe_key,
		          COALESCE(category, ''), status, attempts, next_attempt_at,
		          COALESCE(last_error, ''), COALESCE(provider_message_id, ''),
		          sent_at, created_at, updated_at`

	rows, err := r.db.Query(ctx, query, limit, lease.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []domain.Message
	for rows.Next() {
		msg, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func scanMessage(row pgx.Row) (domain.Message, error) {
	var msg domain.Message
	var payload []byte
	if err := row.Scan(
		&msg.ID, &msg.TemplateKey, &msg.Locale, &msg.ToEmail, &msg.ToName, &payload,
		&msg.DedupeKey, &msg.Category, &msg.Status, &msg.Attempts, &msg.NextAttemptAt,
		&msg.LastError, &msg.ProviderMessageID, &msg.SentAt, &msg.CreatedAt, &msg.UpdatedAt,
	); err != nil {
		return domain.Message{}, err
	}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &msg.Payload); err != nil {
			return domain.Message{}, fmt.Errorf("unmarshal email payload: %w", err)
		}
	}
	if msg.Payload == nil {
		msg.Payload = map[string]any{}
	}
	return msg, nil
}

func (r *PostgresRepository) MarkSent(ctx context.Context, id uuid.UUID, providerMessageID string) error {
	const query = `
		UPDATE email_messages
		SET status = 'sent', sent_at = NOW(), provider_message_id = NULLIF($2, ''),
		    last_error = NULL, updated_at = NOW()
		WHERE id = $1`
	return r.exec(ctx, query, id, providerMessageID)
}

func (r *PostgresRepository) MarkRetry(ctx context.Context, id uuid.UUID, nextAttemptAt time.Time, reason string) error {
	const query = `
		UPDATE email_messages
		SET status = 'pending', next_attempt_at = $2, last_error = $3, updated_at = NOW()
		WHERE id = $1`
	return r.exec(ctx, query, id, nextAttemptAt, truncateError(reason))
}

func (r *PostgresRepository) MarkFailed(ctx context.Context, id uuid.UUID, reason string) error {
	const query = `
		UPDATE email_messages
		SET status = 'failed', last_error = $2, updated_at = NOW()
		WHERE id = $1`
	return r.exec(ctx, query, id, truncateError(reason))
}

func (r *PostgresRepository) MarkSuppressed(ctx context.Context, id uuid.UUID, reason string) error {
	const query = `
		UPDATE email_messages
		SET status = 'suppressed', last_error = $2, updated_at = NOW()
		WHERE id = $1`
	return r.exec(ctx, query, id, truncateError(reason))
}

// MarkDeliveryFailure settles a message the provider later reported as
// undeliverable. Only a message we believe was sent is touched, so a late event
// can't resurrect or overwrite one that has since been retried. A no-match is
// not an error — the provider may report on a message we no longer hold.
func (r *PostgresRepository) MarkDeliveryFailure(ctx context.Context, providerMessageID, reason string) error {
	const query = `
		UPDATE email_messages
		SET status = 'failed', last_error = $2, updated_at = NOW()
		WHERE provider_message_id = $1 AND status = 'sent'`
	_, err := r.db.Exec(ctx, query, providerMessageID, truncateError(reason))
	return err
}

func (r *PostgresRepository) exec(ctx context.Context, query string, args ...any) error {
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrMessageNotFound
	}
	return nil
}

// truncateError keeps a pathological provider error (or a wrapped stack of
// them) from bloating the row; the full text is already in the logs.
func truncateError(reason string) string {
	const maxLen = 1000
	if len(reason) <= maxLen {
		return reason
	}
	return reason[:maxLen]
}
