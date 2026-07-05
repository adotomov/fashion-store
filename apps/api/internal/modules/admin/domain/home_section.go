package domain

import "time"

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
