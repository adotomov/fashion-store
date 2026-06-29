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

type PostgresCategoryRepository struct {
	db *pgxpool.Pool
}

func NewPostgresCategoryRepository(db *pgxpool.Pool) *PostgresCategoryRepository {
	return &PostgresCategoryRepository{db: db}
}

const categoryColumns = `id, name, slug, parent_id, product_type_id,
	thumbnail_bucket, thumbnail_object_key, thumbnail_content_type, thumbnail_size_bytes,
	created_at, updated_at`

func (r *PostgresCategoryRepository) Create(ctx context.Context, category domain.Category) (*domain.Category, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO categories (name, slug, parent_id, product_type_id)
		VALUES ($1, $2, $3, $4)
		RETURNING `+categoryColumns,
		category.Name, category.Slug, category.ParentID, category.ProductTypeID)

	created, err := scanCategory(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, application.ErrSlugConflict
		}
		return nil, err
	}
	return created, nil
}

func (r *PostgresCategoryRepository) List(ctx context.Context) ([]domain.Category, error) {
	rows, err := r.db.Query(ctx, `SELECT `+categoryColumns+` FROM categories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []domain.Category
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		categories = append(categories, *c)
	}
	return categories, rows.Err()
}

func (r *PostgresCategoryRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	row := r.db.QueryRow(ctx, `SELECT `+categoryColumns+` FROM categories WHERE id = $1`, id)

	return scanCategory(row)
}

func (r *PostgresCategoryRepository) Update(ctx context.Context, category domain.Category) (*domain.Category, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE categories SET
			name = $2, parent_id = $3, product_type_id = $4,
			thumbnail_bucket = $5, thumbnail_object_key = $6, thumbnail_content_type = $7, thumbnail_size_bytes = $8,
			updated_at = NOW()
		WHERE id = $1
		RETURNING `+categoryColumns,
		category.ID, category.Name, category.ParentID, category.ProductTypeID,
		category.ThumbnailBucket, category.ThumbnailObjectKey, category.ThumbnailContentType, category.ThumbnailSizeBytes)

	return scanCategory(row)
}

func (r *PostgresCategoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrCategoryNotFound
	}
	return nil
}

func scanCategory(row pgx.Row) (*domain.Category, error) {
	var c domain.Category
	err := row.Scan(
		&c.ID, &c.Name, &c.Slug, &c.ParentID, &c.ProductTypeID,
		&c.ThumbnailBucket, &c.ThumbnailObjectKey, &c.ThumbnailContentType, &c.ThumbnailSizeBytes,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrCategoryNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}
