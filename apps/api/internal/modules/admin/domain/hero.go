package domain

import "time"

type HeroSettings struct {
	Eyebrow           string
	Heading           string
	Subtext           string
	CTAPrimaryLabel   string
	CTAPrimaryURL     string
	CTASecondaryLabel *string
	CTASecondaryURL   *string
	BackgroundBucket       *string
	BackgroundObjectKey    *string
	BackgroundContentType  *string
	BackgroundSizeBytes    *int64
	UpdatedAt         time.Time
}

func (h *HeroSettings) HasBackground() bool {
	return h.BackgroundBucket != nil && *h.BackgroundBucket != "" &&
		h.BackgroundObjectKey != nil && *h.BackgroundObjectKey != ""
}
