package domain

import "testing"

// validAddress is a fully-resolved Speedy address that passes Validate, used as
// a base the tests mutate to exercise individual failure cases.
func validAddress() Address {
	return Address{
		RecipientName: "Jane Doe",
		CountryCode:   "BG",
		CountryID:     100,
		SiteID:        68134,
		City:          "Sofia",
		ComplexID:     1,
		ComplexName:   "Lozenets",
		StreetID:      100,
		StreetName:    "Vitosha",
		StreetNo:      "15",
	}
}

func TestAddress_ValidateFailsWhenRecipientNameMissing(t *testing.T) {
	addr := validAddress()
	addr.RecipientName = ""

	if err := addr.Validate(); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestAddress_ValidateFailsWhenSiteMissing(t *testing.T) {
	addr := validAddress()
	addr.SiteID = 0

	if err := addr.Validate(); err == nil {
		t.Fatal("expected validation error when site (city) is missing, got nil")
	}
}

func TestAddress_ValidateFailsWhenComplexMissing(t *testing.T) {
	addr := validAddress()
	addr.ComplexID = 0

	if err := addr.Validate(); err == nil {
		t.Fatal("expected validation error when complex is missing, got nil")
	}
}

func TestAddress_ValidateFailsWhenStreetMissing(t *testing.T) {
	addr := validAddress()
	addr.StreetID = 0

	if err := addr.Validate(); err == nil {
		t.Fatal("expected validation error when street is missing, got nil")
	}
}

func TestAddress_ValidateFailsWhenCountryCodeNotTwoLetters(t *testing.T) {
	addr := validAddress()
	addr.CountryCode = "BGR"

	if err := addr.Validate(); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestAddress_ValidatePassesWithRequiredFields(t *testing.T) {
	if err := validAddress().Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
