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

type PostgresProductTypeRepository struct {
	db *pgxpool.Pool
}

func NewPostgresProductTypeRepository(db *pgxpool.Pool) *PostgresProductTypeRepository {
	return &PostgresProductTypeRepository{db: db}
}

func (r *PostgresProductTypeRepository) Create(ctx context.Context, productType domain.ProductType) (*domain.ProductType, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO product_types (name, slug)
		VALUES ($1, $2)
		RETURNING id, name, slug, position, created_at, updated_at`,
		productType.Name, productType.Slug)

	created, err := scanProductType(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, application.ErrSlugConflict
		}
		return nil, err
	}
	return created, nil
}

func (r *PostgresProductTypeRepository) List(ctx context.Context) ([]domain.ProductType, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, slug, position, created_at, updated_at
		FROM product_types ORDER BY position, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var productTypes []domain.ProductType
	for rows.Next() {
		t, err := scanProductType(rows)
		if err != nil {
			return nil, err
		}
		productTypes = append(productTypes, *t)
	}
	return productTypes, rows.Err()
}

func (r *PostgresProductTypeRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.ProductType, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, name, slug, position, created_at, updated_at
		FROM product_types WHERE id = $1`, id)

	return scanProductType(row)
}

func (r *PostgresProductTypeRepository) Update(ctx context.Context, productType domain.ProductType) (*domain.ProductType, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE product_types SET name = $2, position = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, slug, position, created_at, updated_at`,
		productType.ID, productType.Name, productType.Position)

	return scanProductType(row)
}

func (r *PostgresProductTypeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM product_types WHERE id = $1`, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return domain.ValidationError("cannot delete a product type while categories are still assigned to it")
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrProductTypeNotFound
	}
	return nil
}

func scanProductType(row pgx.Row) (*domain.ProductType, error) {
	var t domain.ProductType
	err := row.Scan(&t.ID, &t.Name, &t.Slug, &t.Position, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrProductTypeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}
