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

const storeDocumentColumns = `type, locale, bucket, object_key, content_type, size_bytes, filename, content_md, created_at, updated_at`

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
			size_bytes = EXCLUDED.size_bytes, filename = EXCLUDED.filename,
			content_md = NULL, updated_at = NOW()
		RETURNING `+storeDocumentColumns,
		string(d.Type), d.Locale, d.Bucket, d.ObjectKey, d.ContentType, d.SizeBytes, d.Filename)
	return scanStoreDocument(row)
}

// UpsertContent saves inline Markdown content, clearing any GCS file reference.
func (r *PostgresStoreDocumentRepository) UpsertContent(ctx context.Context, docType domain.DocumentType, locale, content string) (*domain.StoreDocument, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO store_documents (type, locale, content_md)
		VALUES ($1, $2, $3)
		ON CONFLICT (type, locale) DO UPDATE SET
			content_md = EXCLUDED.content_md,
			bucket = NULL, object_key = NULL, content_type = NULL, size_bytes = NULL, filename = NULL,
			updated_at = NOW()
		RETURNING `+storeDocumentColumns,
		string(docType), locale, content)
	return scanStoreDocument(row)
}

func (r *PostgresStoreDocumentRepository) Delete(ctx context.Context, docType domain.DocumentType, locale string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM store_documents WHERE type = $1 AND locale = $2`, string(docType), locale)
	return err
}

func scanStoreDocument(row pgx.Row) (*domain.StoreDocument, error) {
	var d domain.StoreDocument
	var docType string
	var bucket, objectKey, contentType, filename *string
	var sizeBytes *int64
	err := row.Scan(&docType, &d.Locale, &bucket, &objectKey, &contentType, &sizeBytes, &filename, &d.ContentMD, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	d.Type = domain.DocumentType(docType)
	if bucket != nil {
		d.Bucket = *bucket
	}
	if objectKey != nil {
		d.ObjectKey = *objectKey
	}
	if contentType != nil {
		d.ContentType = *contentType
	}
	if sizeBytes != nil {
		d.SizeBytes = *sizeBytes
	}
	if filename != nil {
		d.Filename = *filename
	}
	return &d, nil
}
