package infrastructure

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
)

type PostgresAttributeRepository struct {
	db *pgxpool.Pool
}

func NewPostgresAttributeRepository(db *pgxpool.Pool) *PostgresAttributeRepository {
	return &PostgresAttributeRepository{db: db}
}

func (r *PostgresAttributeRepository) Create(ctx context.Context, attribute domain.Attribute) (*domain.Attribute, error) {
	attrType := attribute.Type
	if attrType == "" {
		attrType = domain.AttributeTypeText
	}
	row := r.db.QueryRow(ctx, `
		INSERT INTO attributes (name, type, is_system)
		VALUES ($1, $2, $3)
		RETURNING id, name, type, is_system, created_at, updated_at`,
		attribute.Name, attrType, attribute.IsSystem)

	a, err := scanAttribute(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, application.ErrSlugConflict
		}
		return nil, err
	}
	a.Values = []domain.AttributeValue{}
	return a, nil
}

func (r *PostgresAttributeRepository) List(ctx context.Context) ([]domain.Attribute, error) {
	rows, err := r.db.Query(ctx, `SELECT id, name, type, is_system, created_at, updated_at FROM attributes ORDER BY is_system DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attributes []domain.Attribute
	for rows.Next() {
		a, err := scanAttribute(rows)
		if err != nil {
			return nil, err
		}
		attributes = append(attributes, *a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range attributes {
		values, err := r.valuesFor(ctx, attributes[i].ID)
		if err != nil {
			return nil, err
		}
		attributes[i].Values = values
	}

	return attributes, nil
}

func (r *PostgresAttributeRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Attribute, error) {
	row := r.db.QueryRow(ctx, `SELECT id, name, type, is_system, created_at, updated_at FROM attributes WHERE id = $1`, id)

	a, err := scanAttribute(row)
	if err != nil {
		return nil, err
	}

	values, err := r.valuesFor(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	a.Values = values

	return a, nil
}

func (r *PostgresAttributeRepository) UpdateName(ctx context.Context, id uuid.UUID, name string) (*domain.Attribute, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE attributes SET name = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, type, is_system, created_at, updated_at`,
		id, name)

	a, err := scanAttribute(row)
	if err != nil {
		return nil, err
	}

	values, err := r.valuesFor(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	a.Values = values

	return a, nil
}

func (r *PostgresAttributeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM attributes WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrAttributeNotFound
	}
	return nil
}

func (r *PostgresAttributeRepository) AddValue(ctx context.Context, attributeID uuid.UUID, value string, colorHex *string) (*domain.AttributeValue, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO attribute_values (attribute_id, value, color_hex)
		VALUES ($1, $2, $3)
		RETURNING id, attribute_id, value, color_hex, created_at`,
		attributeID, value, colorHex)

	v, err := scanAttributeValue(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, application.ErrSlugConflict
		}
		return nil, err
	}
	return v, nil
}

func (r *PostgresAttributeRepository) DeleteValue(ctx context.Context, attributeID, valueID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM attribute_values WHERE id = $1 AND attribute_id = $2`, valueID, attributeID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrAttributeValueNotFound
	}
	return nil
}

func (r *PostgresAttributeRepository) valuesFor(ctx context.Context, attributeID uuid.UUID) ([]domain.AttributeValue, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, attribute_id, value, color_hex, created_at FROM attribute_values
		WHERE attribute_id = $1 ORDER BY value`, attributeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := []domain.AttributeValue{}
	for rows.Next() {
		v, err := scanAttributeValue(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, *v)
	}
	return values, rows.Err()
}

func scanAttribute(row pgx.Row) (*domain.Attribute, error) {
	var a domain.Attribute
	err := row.Scan(&a.ID, &a.Name, &a.Type, &a.IsSystem, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrAttributeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func scanAttributeValue(row pgx.Row) (*domain.AttributeValue, error) {
	var v domain.AttributeValue
	err := row.Scan(&v.ID, &v.AttributeID, &v.Value, &v.ColorHex, &v.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrAttributeValueNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}
