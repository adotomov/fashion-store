package application

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/i18n/domain"
)

// ResolveInput carries the per-request signals used to pick a display locale, in
// the module's precedence order: a signed-in user's stored locale (their Google
// account language), then the request's geo country, then the browser's
// Accept-Language, then the store default / English base. An explicit manual
// choice sits above all of these and is applied on the client, so it is not a
// signal here.
type ResolveInput struct {
	UserLocale     string
	GeoCountry     string
	AcceptLanguage string
}

// Resolve picks the best enabled language for the request. It never yields a
// missing locale: an unmatched request settles on the default display language,
// or the English base if that is somehow unavailable.
func (s *LanguageService) Resolve(ctx context.Context, in ResolveInput) (string, error) {
	enabled, err := s.ListEnabled(ctx)
	if err != nil {
		return "", err
	}

	byCode := make(map[string]bool, len(enabled))
	byCountry := make(map[string]string, len(enabled))
	def := domain.DefaultLocale
	for _, l := range enabled {
		byCode[l.Code] = true
		if l.CountryCode != "" {
			byCountry[strings.ToUpper(l.CountryCode)] = l.Code
		}
		if l.IsDefault {
			def = l.Code
		}
	}

	// 1. Signed-in user's own preference (Google account) wins over location.
	if code := domain.NormalizeLanguage(in.UserLocale); code != "" && byCode[code] {
		return code, nil
	}
	// 2. Location: the language mapped to the visitor's country.
	if country := strings.ToUpper(strings.TrimSpace(in.GeoCountry)); country != "" {
		if code, ok := byCountry[country]; ok {
			return code, nil
		}
	}
	// 3. Browser preference, in descending q-value order.
	for _, tag := range parseAcceptLanguage(in.AcceptLanguage) {
		if code := domain.NormalizeLanguage(tag); byCode[code] {
			return code, nil
		}
	}
	// 4. Store default display language, else the English base.
	if byCode[def] {
		return def, nil
	}
	return domain.DefaultLocale, nil
}

// parseAcceptLanguage returns the language tags from an Accept-Language header
// ordered by descending q-value (default q=1). Malformed entries and the "*"
// wildcard are skipped.
func parseAcceptLanguage(header string) []string {
	if strings.TrimSpace(header) == "" {
		return nil
	}
	type weighted struct {
		tag string
		q   float64
	}
	var items []weighted
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		tag, q := part, 1.0
		if i := strings.Index(part, ";"); i >= 0 {
			tag = strings.TrimSpace(part[:i])
			params := strings.ToLower(part[i+1:])
			if j := strings.Index(params, "q="); j >= 0 {
				if v, err := strconv.ParseFloat(strings.TrimSpace(params[j+2:]), 64); err == nil {
					q = v
				}
			}
		}
		if tag == "" || tag == "*" {
			continue
		}
		items = append(items, weighted{tag: tag, q: q})
	}
	sort.SliceStable(items, func(a, b int) bool { return items[a].q > items[b].q })
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.tag)
	}
	return out
}
