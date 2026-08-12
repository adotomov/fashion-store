package application

import (
	"context"
	"testing"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/i18n/domain"
)

// fakeLangRepo serves a fixed language set; only List is exercised by Resolve.
type fakeLangRepo struct{ langs []domain.Language }

func (f *fakeLangRepo) List(context.Context) ([]domain.Language, error) { return f.langs, nil }
func (f *fakeLangRepo) Get(context.Context, string) (*domain.Language, error) {
	return nil, domain.ErrLanguageNotFound
}
func (f *fakeLangRepo) Create(_ context.Context, l domain.Language) (*domain.Language, error) {
	return &l, nil
}
func (f *fakeLangRepo) SetEnabled(context.Context, string, bool) (*domain.Language, error) {
	return nil, nil
}
func (f *fakeLangRepo) SetDefault(context.Context, string) (*domain.Language, error) { return nil, nil }
func (f *fakeLangRepo) SetCountry(context.Context, string, string) (*domain.Language, error) {
	return nil, nil
}
func (f *fakeLangRepo) Delete(context.Context, string) error { return nil }

func TestResolvePrecedence(t *testing.T) {
	svc := NewLanguageService(&fakeLangRepo{langs: []domain.Language{
		{Code: "en", Name: "English", IsDefault: true, Enabled: true},
		{Code: "bg", Name: "Български", Enabled: true, CountryCode: "BG"},
		{Code: "de", Name: "Deutsch", Enabled: false, CountryCode: "DE"}, // disabled → never chosen
	}})
	ctx := context.Background()

	cases := []struct {
		name string
		in   ResolveInput
		want string
	}{
		{"signed-in Google bg beats geo DE", ResolveInput{UserLocale: "bg-BG", GeoCountry: "DE", AcceptLanguage: "de"}, "bg"},
		{"anonymous geo BG", ResolveInput{GeoCountry: "BG", AcceptLanguage: "en-US,en;q=0.9"}, "bg"},
		{"anonymous geo DE, no browser match → default", ResolveInput{GeoCountry: "DE", AcceptLanguage: "de,fr;q=0.8"}, "en"},
		{"anonymous geo DE, browser prefers bg", ResolveInput{GeoCountry: "DE", AcceptLanguage: "bg,en;q=0.8"}, "bg"},
		{"unknown user locale falls through to geo", ResolveInput{UserLocale: "fr", GeoCountry: "BG"}, "bg"},
		{"no signal → default", ResolveInput{}, "en"},
		{"disabled language not chosen despite geo match", ResolveInput{GeoCountry: "DE"}, "en"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := svc.Resolve(ctx, c.in)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}
