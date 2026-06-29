package application

import (
	"context"
	"io"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/admin/domain"
)

// MediaStorage isolates the GCS-compatible object storage vendor from
// application logic. Implemented in internal/platform/storage — same port
// the catalog module uses for product media and category thumbnails.
type MediaStorage interface {
	EnsureBucket(ctx context.Context, bucket string) error
	Upload(ctx context.Context, bucket, objectKey, contentType string, content io.Reader) (sizeBytes int64, err error)
	Open(ctx context.Context, bucket, objectKey string) (reader io.ReadCloser, contentType string, err error)
	Delete(ctx context.Context, bucket, objectKey string) error
}

// StoreSettingsRepository persists the single store_settings row. Get
// returns that row (it always exists — seeded by migration); Update
// persists changes to it.
type StoreSettingsRepository interface {
	Get(ctx context.Context) (*domain.StoreSettings, error)
	Update(ctx context.Context, settings domain.StoreSettings) (*domain.StoreSettings, error)
}

// StoreAddressRepository persists the zero-or-more addresses for the store
// settings singleton — multi-location stores have more than one.
type StoreAddressRepository interface {
	List(ctx context.Context, storeSettingsID uuid.UUID) ([]domain.StoreAddress, error)
	Create(ctx context.Context, address domain.StoreAddress) (*domain.StoreAddress, error)
	Update(ctx context.Context, address domain.StoreAddress) (*domain.StoreAddress, error)
	Delete(ctx context.Context, id uuid.UUID) error
	// ClearDefault unsets is_default on every address for the store, used
	// before setting a new one so exactly one default ever exists.
	ClearDefault(ctx context.Context, storeSettingsID uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*domain.StoreAddress, error)
}

// StoreDocumentRepository persists legal document uploads keyed by
// (type, locale) — each language has its own Terms/Privacy file.
type StoreDocumentRepository interface {
	List(ctx context.Context, docType domain.DocumentType) ([]domain.StoreDocument, error)
	Get(ctx context.Context, docType domain.DocumentType, locale string) (*domain.StoreDocument, error)
	Upsert(ctx context.Context, doc domain.StoreDocument) (*domain.StoreDocument, error)
	Delete(ctx context.Context, docType domain.DocumentType, locale string) error
}
