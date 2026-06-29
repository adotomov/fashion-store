package domain

import "time"

// DocumentType identifies which legal document an upload/serve/delete
// operation targets.
type DocumentType string

const (
	DocumentTypeTerms   DocumentType = "terms"
	DocumentTypePrivacy DocumentType = "privacy"
)

// StoreDocument is one legal document upload for one (type, locale) pair —
// e.g. the German Terms of Service is a separate row from the English one.
type StoreDocument struct {
	Type        DocumentType
	Locale      string
	Bucket      string
	ObjectKey   string
	ContentType string
	SizeBytes   int64
	Filename    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
