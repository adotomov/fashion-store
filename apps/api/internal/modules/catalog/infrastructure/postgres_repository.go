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

type PostgresCatalogRepository struct {
	db *pgxpool.Pool
}

func NewPostgresCatalogRepository(db *pgxpool.Pool) *PostgresCatalogRepository {
	return &PostgresCatalogRepository{db: db}
}

func (r *PostgresCatalogRepository) Create(ctx context.Context, catalog domain.Catalog) (*domain.Catalog, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO catalogs (name, slug, description, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, slug, COALESCE(description, ''), status, created_at, updated_at`,
		catalog.Name, catalog.Slug, catalog.Description, catalog.Status)

	created, err := scanCatalog(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, application.ErrSlugConflict
		}
		return nil, err
	}
	return created, nil
}

func (r *PostgresCatalogRepository) List(ctx context.Context) ([]domain.Catalog, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, slug, COALESCE(description, ''), status, created_at, updated_at
		FROM catalogs ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var catalogs []domain.Catalog
	for rows.Next() {
		c, err := scanCatalog(rows)
		if err != nil {
			return nil, err
		}
		catalogs = append(catalogs, *c)
	}
	return catalogs, rows.Err()
}

func (r *PostgresCatalogRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Catalog, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, name, slug, COALESCE(description, ''), status, created_at, updated_at
		FROM catalogs WHERE id = $1`, id)

	return scanCatalog(row)
}

func (r *PostgresCatalogRepository) Update(ctx context.Context, catalog domain.Catalog) (*domain.Catalog, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE catalogs SET name = $2, description = $3, status = $4, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, slug, COALESCE(description, ''), status, created_at, updated_at`,
		catalog.ID, catalog.Name, catalog.Description, catalog.Status)

	return scanCatalog(row)
}

func (r *PostgresCatalogRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM catalogs WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrCatalogNotFound
	}
	return nil
}

func scanCatalog(row pgx.Row) (*domain.Catalog, error) {
	var c domain.Catalog
	var status string
	err := row.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &status, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrCatalogNotFound
	}
	if err != nil {
		return nil, err
	}
	c.Status = domain.Status(status)
	return &c, nil
}
