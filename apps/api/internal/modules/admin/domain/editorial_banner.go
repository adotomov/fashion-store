package domain

import "time"

// EditorialBanner is the singleton "Shop the Look" banner rendered mid-home-page.
// It mirrors HeroSettings but carries an Enabled flag (the section is hidden
// until an admin turns it on) and a single CTA.
type EditorialBanner struct {
	Enabled  bool
	Eyebrow  string
	Heading  string
	Subtext  string
	CTALabel string
	CTAURL   string

	ImageBucket      *string
	ImageObjectKey   *string
	ImageContentType *string
	ImageSizeBytes   *int64

	UpdatedAt time.Time
}

func (b *EditorialBanner) HasImage() bool {
	return b.ImageBucket != nil && *b.ImageBucket != "" &&
		b.ImageObjectKey != nil && *b.ImageObjectKey != ""
}
