package domain

import (
	"time"

	"github.com/google/uuid"
)

// HomeSectionID constants match the rows seeded by migration.
const (
	HomeSectionSpotlights     = "spotlights"
	HomeSectionRecommended    = "recommended"
	HomeSectionOnSale         = "on_sale"
	HomeSectionBestInCategory = "best_in_category"
)

// CuratedSections require admin-picked product lists; the rest are auto-populated.
var CuratedSections = map[string]bool{
	HomeSectionSpotlights:  true,
	HomeSectionRecommended: true,
}

type HomeSection struct {
	ID        string
	Enabled   bool
	Eyebrow   string
	Heading   string
	UpdatedAt time.Time
}

// SectionCategoryGroup is one curated category within a section (e.g. "Best in
// its category"): the category plus its ordered, hand-picked product list.
// Groups themselves are ordered by the caller (their slice position).
type SectionCategoryGroup struct {
	CategoryID uuid.UUID
	ProductIDs []uuid.UUID
}
