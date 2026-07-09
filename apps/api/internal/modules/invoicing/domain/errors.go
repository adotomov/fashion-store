package domain

import "errors"

var (
	ErrInvoiceNotFound       = errors.New("invoice not found")
	ErrInvoiceAlreadyExists  = errors.New("an invoice already exists for this order")
	ErrSettingsIncomplete    = errors.New("invoice settings are incomplete: company EIK and NRA store number must be configured")
	ErrCannotStornoAStorno   = errors.New("cannot issue a storno on a storno document")
	ErrInvalidTaxGroup       = errors.New("tax group identifier must be one of А–Ж and VAT rate must be between 0 and 100")
)

// TaxGroupIdentifiers are the valid Bulgarian VAT fiscal group letters.
var TaxGroupIdentifiers = []string{"А", "Б", "В", "Г", "Д", "Е", "Ж"}

// ValidTaxGroupIdentifier reports whether id is one of the allowed А–Ж letters.
func ValidTaxGroupIdentifier(id string) bool {
	for _, v := range TaxGroupIdentifiers {
		if v == id {
			return true
		}
	}
	return false
}
