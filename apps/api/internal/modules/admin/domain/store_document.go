package domain

import "time"

// DocumentType identifies which legal document an upload/serve/delete
// operation targets.
type DocumentType string

const (
	DocumentTypeTerms   DocumentType = "terms"
	DocumentTypePrivacy DocumentType = "privacy"
)

// StoreDocument is one legal document for one (type, locale) pair.
// Either ContentMD is set (inline Markdown editor flow) or the GCS
// fields are set (legacy file-upload flow); never both at once.
type StoreDocument struct {
	Type        DocumentType
	Locale      string
	ContentMD   *string
	Bucket      string
	ObjectKey   string
	ContentType string
	SizeBytes   int64
	Filename    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
