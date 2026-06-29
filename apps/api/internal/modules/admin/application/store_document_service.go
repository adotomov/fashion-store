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
	if docType != domain.DocumentTypeTerms && docType != domain.DocumentTypePrivacy {
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

// Open returns the document for a (type, locale) pair, falling back to the
// default English document if the requested locale hasn't been uploaded.
func (s *StoreDocumentService) Open(ctx context.Context, docType domain.DocumentType, locale string) (io.ReadCloser, string, error) {
	doc, err := s.repo.Get(ctx, docType, locale)
	if err != nil || doc == nil {
		doc, err = s.repo.Get(ctx, docType, "en")
		if err != nil {
			return nil, "", err
		}
		if doc == nil {
			return nil, "", domain.ErrDocumentNotFound
		}
	}
	return s.storage.Open(ctx, doc.Bucket, doc.ObjectKey)
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
