package domain

import "errors"

var (
	ErrLogoNotFound              = errors.New("store logo not found")
	ErrDocumentNotFound          = errors.New("document not found")
	ErrInvalidDocumentType       = errors.New("invalid document type")
	ErrAddressNotFound           = errors.New("store address not found")
	ErrHeroBackgroundNotFound    = errors.New("hero background image not found")
	ErrEditorialBannerImageNotFound = errors.New("editorial banner image not found")
)
