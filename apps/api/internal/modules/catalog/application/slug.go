package application

import (
	"regexp"
	"strings"
)

var (
	nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)
	trimHyphens     = regexp.MustCompile(`^-+|-+$`)
)

// slugify produces a URL-friendly slug candidate from a catalog name. It
// does not guarantee uniqueness; the caller retries with randomSuffix on
// conflict.
func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = nonAlphanumeric.ReplaceAllString(s, "-")
	s = trimHyphens.ReplaceAllString(s, "")
	if s == "" {
		s = "catalog"
	}
	return s
}
