package domain

import (
	"errors"
	"strings"

	"github.com/google/uuid"
)

// NormalizeLanguage reduces a BCP-47 locale tag (e.g. "en-GB", "bg", "pt_BR") to
// its lowercase primary language subtag ("en", "bg", "pt"). Empty in, empty out.
// It does not check whether the language is enabled — callers layer that on.
func NormalizeLanguage(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	if i := strings.IndexAny(s, "-_"); i >= 0 {
		s = s[:i]
	}
	return s
}

type Language struct {
	Code string
	Name string
	// IsDefault marks the store's default DISPLAY language — what a visitor sees
	// when no stronger signal (explicit choice, Google account, geo, browser)
	// applies. Distinct from DefaultLocale below, which is the immutable English
	// content/fallback base.
	IsDefault bool
	Enabled   bool
	// CountryCode is the ISO-3166 alpha-2 country (e.g. "BG") this language serves,
	// used to pick a language from the request's geo region. Empty = not geo-mapped.
	CountryCode string
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
	ErrLanguageNotEnabled        = errors.New("a disabled language cannot be made the default")
)
