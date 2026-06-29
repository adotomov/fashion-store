package domain

import "testing"

func TestAddress_ValidateFailsWhenRecipientNameMissing(t *testing.T) {
	addr := Address{
		Line1:       "Main St 1",
		City:        "Sofia",
		PostalCode:  "1000",
		CountryCode: "BG",
	}

	if err := addr.Validate(); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestAddress_ValidateFailsWhenCountryCodeNotTwoLetters(t *testing.T) {
	addr := Address{
		RecipientName: "Jane Doe",
		Line1:         "Main St 1",
		City:          "Sofia",
		PostalCode:    "1000",
		CountryCode:   "BGR",
	}

	if err := addr.Validate(); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestAddress_ValidatePassesWithRequiredFields(t *testing.T) {
	addr := Address{
		RecipientName: "Jane Doe",
		Line1:         "Main St 1",
		City:          "Sofia",
		PostalCode:    "1000",
		CountryCode:   "BG",
	}

	if err := addr.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
