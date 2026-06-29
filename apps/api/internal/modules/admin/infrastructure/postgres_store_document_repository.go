package infrastructure

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/admin/domain"
)

type PostgresStoreDocumentRepository struct {
	db *pgxpool.Pool
}

func NewPostgresStoreDocumentRepository(db *pgxpool.Pool) *PostgresStoreDocumentRepository {
	return &PostgresStoreDocumentRepository{db: db}
}

const storeDocumentColumns = `type, locale, bucket, object_key, content_type, size_bytes, filename, created_at, updated_at`

func (r *PostgresStoreDocumentRepository) List(ctx context.Context, docType domain.DocumentType) ([]domain.StoreDocument, error) {
	rows, err := r.db.Query(ctx, `SELECT `+storeDocumentColumns+` FROM store_documents WHERE type = $1 ORDER BY locale`, string(docType))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.StoreDocument
	for rows.Next() {
		d, err := scanStoreDocument(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (r *PostgresStoreDocumentRepository) Get(ctx context.Context, docType domain.DocumentType, locale string) (*domain.StoreDocument, error) {
	row := r.db.QueryRow(ctx, `SELECT `+storeDocumentColumns+` FROM store_documents WHERE type = $1 AND locale = $2`, string(docType), locale)
	doc, err := scanStoreDocument(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return doc, err
}

func (r *PostgresStoreDocumentRepository) Upsert(ctx context.Context, d domain.StoreDocument) (*domain.StoreDocument, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO store_documents (type, locale, bucket, object_key, content_type, size_bytes, filename)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (type, locale) DO UPDATE SET
			bucket = EXCLUDED.bucket, object_key = EXCLUDED.object_key, content_type = EXCLUDED.content_type,
			size_bytes = EXCLUDED.size_bytes, filename = EXCLUDED.filename, updated_at = NOW()
		RETURNING `+storeDocumentColumns,
		string(d.Type), d.Locale, d.Bucket, d.ObjectKey, d.ContentType, d.SizeBytes, d.Filename)
	return scanStoreDocument(row)
}

func (r *PostgresStoreDocumentRepository) Delete(ctx context.Context, docType domain.DocumentType, locale string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM store_documents WHERE type = $1 AND locale = $2`, string(docType), locale)
	return err
}

func scanStoreDocument(row pgx.Row) (*domain.StoreDocument, error) {
	var d domain.StoreDocument
	var docType string
	err := row.Scan(&docType, &d.Locale, &d.Bucket, &d.ObjectKey, &d.ContentType, &d.SizeBytes, &d.Filename, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	d.Type = domain.DocumentType(docType)
	return &d, nil
}
