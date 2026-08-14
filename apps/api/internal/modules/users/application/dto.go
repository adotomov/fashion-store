package application

type UpdateProfileInput struct {
	FullName *string
	Phone    *string
}

// AddressInput is the full structured address a client submits when creating or
// updating a saved address. A Speedy-resolved address is atomic (the city,
// complex and street codes must be consistent), so update replaces the whole
// address rather than patching individual fields.
type AddressInput struct {
	Label         string
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

	IsDefault bool
}

type (
	AddAddressInput    = AddressInput
	UpdateAddressInput = AddressInput
)
