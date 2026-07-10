package domain

import "time"

// DocumentType identifies which legal document an upload/serve/delete
// operation targets.
type DocumentType string

const (
	DocumentTypeTerms    DocumentType = "terms"
	DocumentTypePrivacy  DocumentType = "privacy"
	DocumentTypeFAQ      DocumentType = "faq"
	DocumentTypeShipping DocumentType = "shipping"
)

// IsValid reports whether t is a recognised document type.
func (t DocumentType) IsValid() bool {
	switch t {
	case DocumentTypeTerms, DocumentTypePrivacy, DocumentTypeFAQ, DocumentTypeShipping:
		return true
	default:
		return false
	}
}

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
