package domain

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID            uuid.UUID
	Name          string
	Slug          string
	ParentID      *uuid.UUID
	ProductTypeID uuid.UUID
	// InternalIdentifier is an admin-facing code (e.g. "DR-01") used as the
	// fixed prefix when composing variant SKUs for products in this category.
	// Empty string means none has been assigned yet.
	InternalIdentifier string
	// Thumbnail* describe an optional image shown in the storefront nav
	// menu, stored in object storage (GCS/fakegcs) the same way product
	// media is — nil ThumbnailObjectKey means none has been uploaded yet.
	ThumbnailBucket      *string
	ThumbnailObjectKey   *string
	ThumbnailContentType *string
	ThumbnailSizeBytes   *int64
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (c Category) HasThumbnail() bool {
	return c.ThumbnailObjectKey != nil && *c.ThumbnailObjectKey != ""
}
