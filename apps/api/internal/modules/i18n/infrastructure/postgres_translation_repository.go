package infrastructure

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/i18n/domain"
)

type PostgresTranslationRepository struct {
	db *pgxpool.Pool
}

func NewPostgresTranslationRepository(db *pgxpool.Pool) *PostgresTranslationRepository {
	return &PostgresTranslationRepository{db: db}
}

func (r *PostgresTranslationRepository) Get(ctx context.Context, entityType string, entityID uuid.UUID, locale string) (map[string]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT field, value FROM translations
		WHERE entity_type = $1 AND entity_id = $2 AND locale = $3`,
		entityType, entityID, locale)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]string{}
	for rows.Next() {
		var field, value string
		if err := rows.Scan(&field, &value); err != nil {
			return nil, err
		}
		out[field] = value
	}
	return out, rows.Err()
}

func (r *PostgresTranslationRepository) ListByEntityType(ctx context.Context, entityType string, locale string) (map[uuid.UUID]map[string]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT entity_id, field, value FROM translations
		WHERE entity_type = $1 AND locale = $2`,
		entityType, locale)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[uuid.UUID]map[string]string{}
	for rows.Next() {
		var entityID uuid.UUID
		var field, value string
		if err := rows.Scan(&entityID, &field, &value); err != nil {
			return nil, err
		}
		if out[entityID] == nil {
			out[entityID] = map[string]string{}
		}
		out[entityID][field] = value
	}
	return out, rows.Err()
}

func (r *PostgresTranslationRepository) Upsert(ctx context.Context, t domain.Translation) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO translations (entity_type, entity_id, locale, field, value)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (entity_type, entity_id, locale, field)
		DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
		t.EntityType, t.EntityID, t.Locale, t.Field, t.Value)
	return err
}

func (r *PostgresTranslationRepository) Delete(ctx context.Context, entityType string, entityID uuid.UUID, locale, field string) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM translations
		WHERE entity_type = $1 AND entity_id = $2 AND locale = $3 AND field = $4`,
		entityType, entityID, locale, field)
	return err
}

func (r *PostgresTranslationRepository) DeleteEntity(ctx context.Context, entityType string, entityID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM translations WHERE entity_type = $1 AND entity_id = $2`, entityType, entityID)
	return err
}
