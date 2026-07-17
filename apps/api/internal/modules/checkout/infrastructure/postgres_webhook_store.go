package infrastructure

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/checkout/application"
)

// PostgresWebhookEventStore is the idempotency ledger for inbound Revolut
// webhooks, backed by the payment_webhook_events table.
type PostgresWebhookEventStore struct {
	db *pgxpool.Pool
}

func NewPostgresWebhookEventStore(db *pgxpool.Pool) *PostgresWebhookEventStore {
	return &PostgresWebhookEventStore{db: db}
}

func (s *PostgresWebhookEventStore) Seen(ctx context.Context, eventID string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM payment_webhook_events WHERE id = $1)`, eventID).Scan(&exists)
	return exists, err
}

func (s *PostgresWebhookEventStore) Record(ctx context.Context, event application.WebhookEvent) error {
	var providerOrderID *string
	if event.ProviderOrderID != "" {
		providerOrderID = &event.ProviderOrderID
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO payment_webhook_events (id, event_type, provider_order_id, payload)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO NOTHING`,
		event.ID, event.Type, providerOrderID, event.RawPayload)
	return err
}
