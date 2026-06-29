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
	LogoBucket         *string
	LogoObjectKey      *string
	LogoContentType    *string
	LogoSizeBytes      *int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (s StoreSettings) HasLogo() bool {
	return s.LogoObjectKey != nil && *s.LogoObjectKey != ""
}
