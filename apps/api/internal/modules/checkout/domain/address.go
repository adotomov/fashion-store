package domain

// Address is the customer-supplied shipping/billing address for a single
// checkout — snapshotted onto the order rather than referencing the saved
// address book, since a signed-in customer might check out with an address
// they never bother to save. It is Speedy-resolved: city, complex (кв./жк.)
// and street are location codes plus display names. Bulgaria only for now.
type Address struct {
	RecipientName string
	Phone         string

	CountryCode string
	CountryID   int64
	SiteID      int64
	City        string
	PostCode    string
	ComplexID   int64
	ComplexName string
	StreetID    int64
	StreetName  string
	StreetNo    string
	BlockNo     string
	EntranceNo  string
	FloorNo     string
	ApartmentNo string
}

func (a Address) Validate() error {
	if a.RecipientName == "" {
		return ValidationError("recipient_name is required")
	}
	if a.SiteID <= 0 {
		return ValidationError("city (site) is required")
	}
	if a.ComplexID <= 0 {
		return ValidationError("complex (кв./жк.) is required")
	}
	if a.StreetID <= 0 {
		return ValidationError("street is required")
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
