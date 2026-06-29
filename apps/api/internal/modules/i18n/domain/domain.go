package domain

import (
	"errors"

	"github.com/google/uuid"
)

type Language struct {
	Code      string
	Name      string
	IsDefault bool
	Enabled   bool
}

// DefaultLocale is the store's always-present base language. Catalog and
// store-settings rows hold their English content directly; every other
// locale's content lives in Translation rows layered on top.
const DefaultLocale = "en"

type Translation struct {
	EntityType string
	EntityID   uuid.UUID
	Locale     string
	Field      string
	Value      string
}

type UIString struct {
	Key    string
	Locale string
	Value  string
}

var (
	ErrLanguageNotFound          = errors.New("language not found")
	ErrLanguageAlreadyExists     = errors.New("language already exists")
	ErrCannotModifyDefaultLocale = errors.New("the default language cannot be disabled or removed")
	ErrInvalidLanguageCode       = errors.New("invalid language code")
)
