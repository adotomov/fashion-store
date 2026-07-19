package domain

// Address is the customer-supplied shipping/billing address for a single
// checkout — snapshotted onto the order rather than referencing the saved
// address book, since a signed-in customer might check out with an address
// they never bother to save.
type Address struct {
	RecipientName string
	Phone         string
	Line1         string
	Line2         string
	City          string
	Region        string
	PostalCode    string
	CountryCode   string
}

func (a Address) Validate() error {
	if a.RecipientName == "" {
		return ValidationError("recipient_name is required")
	}
	if a.Line1 == "" {
		return ValidationError("line1 is required")
	}
	if a.City == "" {
		return ValidationError("city is required")
	}
	if a.PostalCode == "" {
		return ValidationError("postal_code is required")
	}
	if len(a.CountryCode) != 2 {
		return ValidationError("country_code must be a 2-letter ISO code")
	}
	return nil
}

type Contact struct {
	FullName string
	Email    string
	Phone    string
}

func (c Contact) Validate() error {
	if c.FullName == "" {
		return ValidationError("full_name is required")
	}
	if c.Email == "" {
		return ValidationError("email is required")
	}
	if c.Phone == "" {
		return ValidationError("phone is required")
	}
	return nil
}
