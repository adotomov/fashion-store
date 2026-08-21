package domain

import (
	"time"

	"github.com/google/uuid"
)

// StoreSettings is a singleton: exactly one row exists in the database,
// seeded by migration and only ever updated (never created or deleted) by
// application code. Always-English fields (name, legal entity, locale,
// currency, contact details) live here directly; addresses and legal
// documents live in their own tables to support multiple addresses and
// per-language document uploads.
type StoreSettings struct {
	ID                 uuid.UUID
	StoreName          string
	LegalEntityName    *string
	Locale             string
	Currency           string
	ContactEmail       *string
	ContactPhone       *string
	CompanyDescription *string
	FacebookURL        *string
	InstagramURL       *string
	OpeningHours       *string
	LogoBucket         *string
	LogoObjectKey      *string
	LogoContentType    *string
	LogoSizeBytes      *int64
	// About Us cover photo (public About page) and Contact Us store photo
	// (public Contact page) — nullable file references, same shape as the logo.
	AboutCoverBucket      *string
	AboutCoverObjectKey   *string
	AboutCoverContentType *string
	AboutCoverSizeBytes   *int64
	StoreImageBucket      *string
	StoreImageObjectKey   *string
	StoreImageContentType *string
	StoreImageSizeBytes   *int64
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (s StoreSettings) HasLogo() bool {
	return s.LogoObjectKey != nil && *s.LogoObjectKey != ""
}

func (s StoreSettings) HasAboutCover() bool {
	return s.AboutCoverObjectKey != nil && *s.AboutCoverObjectKey != ""
}

func (s StoreSettings) HasStoreImage() bool {
	return s.StoreImageObjectKey != nil && *s.StoreImageObjectKey != ""
}
