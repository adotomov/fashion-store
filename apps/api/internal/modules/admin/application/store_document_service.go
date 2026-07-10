package application

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/admin/domain"
)

type StoreDocumentService struct {
	repo    StoreDocumentRepository
	storage MediaStorage
	bucket  string
}

func NewStoreDocumentService(repo StoreDocumentRepository, storage MediaStorage, bucket string) *StoreDocumentService {
	return &StoreDocumentService{repo: repo, storage: storage, bucket: bucket}
}

func (s *StoreDocumentService) List(ctx context.Context, docType domain.DocumentType) ([]domain.StoreDocument, error) {
	return s.repo.List(ctx, docType)
}

func (s *StoreDocumentService) Upload(ctx context.Context, docType domain.DocumentType, locale, filename, contentType string, content io.Reader) (*domain.StoreDocument, error) {
	if !docType.IsValid() {
		return nil, domain.ErrInvalidDocumentType
	}
	if err := s.storage.EnsureBucket(ctx, s.bucket); err != nil {
		return nil, err
	}

	if existing, _ := s.repo.Get(ctx, docType, locale); existing != nil {
		_ = s.storage.Delete(ctx, existing.Bucket, existing.ObjectKey)
	}

	objectKey := fmt.Sprintf("store-settings/documents/%s/%s/%s-%s", docType, locale, uuid.NewString(), filename)
	sizeBytes, err := s.storage.Upload(ctx, s.bucket, objectKey, contentType, content)
	if err != nil {
		return nil, err
	}

	return s.repo.Upsert(ctx, domain.StoreDocument{
		Type:        docType,
		Locale:      locale,
		Bucket:      s.bucket,
		ObjectKey:   objectKey,
		ContentType: contentType,
		SizeBytes:   sizeBytes,
		Filename:    filename,
	})
}

// Open returns the file bytes for a GCS-stored document, falling back to English.
func (s *StoreDocumentService) Open(ctx context.Context, docType domain.DocumentType, locale string) (io.ReadCloser, string, error) {
	doc, err := s.repo.Get(ctx, docType, locale)
	if err != nil || doc == nil || doc.ObjectKey == "" {
		doc, err = s.repo.Get(ctx, docType, "en")
		if err != nil {
			return nil, "", err
		}
		if doc == nil || doc.ObjectKey == "" {
			return nil, "", domain.ErrDocumentNotFound
		}
	}
	return s.storage.Open(ctx, doc.Bucket, doc.ObjectKey)
}

// SaveContent persists inline Markdown text for the given (type, locale),
// removing any GCS file that was previously uploaded for that locale.
func (s *StoreDocumentService) SaveContent(ctx context.Context, docType domain.DocumentType, locale, content string) (*domain.StoreDocument, error) {
	if !docType.IsValid() {
		return nil, domain.ErrInvalidDocumentType
	}
	if existing, _ := s.repo.Get(ctx, docType, locale); existing != nil && existing.ObjectKey != "" {
		_ = s.storage.Delete(ctx, existing.Bucket, existing.ObjectKey)
	}
	return s.repo.UpsertContent(ctx, docType, locale, content)
}

// GetContent returns the Markdown text for a (type, locale), falling back to English.
func (s *StoreDocumentService) GetContent(ctx context.Context, docType domain.DocumentType, locale string) (string, error) {
	resolve := func(loc string) (string, bool) {
		doc, err := s.repo.Get(ctx, docType, loc)
		if err != nil || doc == nil || doc.ContentMD == nil {
			return "", false
		}
		return *doc.ContentMD, true
	}
	if content, ok := resolve(locale); ok {
		return content, nil
	}
	if content, ok := resolve("en"); ok {
		return content, nil
	}
	return "", domain.ErrDocumentNotFound
}

func (s *StoreDocumentService) Delete(ctx context.Context, docType domain.DocumentType, locale string) error {
	doc, err := s.repo.Get(ctx, docType, locale)
	if err != nil {
		return err
	}
	if doc != nil {
		_ = s.storage.Delete(ctx, doc.Bucket, doc.ObjectKey)
	}
	return s.repo.Delete(ctx, docType, locale)
}
