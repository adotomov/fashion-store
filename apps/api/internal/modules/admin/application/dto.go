package application

// UpdateStoreSettingsInput carries partial updates to the store settings
// singleton — nil fields are left unchanged.
type UpdateStoreSettingsInput struct {
	StoreName          *string
	LegalEntityName    *string
	Locale             *string
	Currency           *string
	ContactEmail       *string
	ContactPhone       *string
	CompanyDescription *string
	FacebookURL        *string
	InstagramURL       *string
	OpeningHours       *string
	MapLocation        *string
	// FreeShippingThresholdMinor is in EUR minor units; zero makes shipping free on every order.
	FreeShippingThresholdMinor *int64
}

// UpsertStoreAddressInput carries the fields for creating or updating a
// store address.
type UpsertStoreAddressInput struct {
	Label      string
	Line1      string
	Line2      *string
	City       *string
	Region     *string
	PostalCode *string
	Country    *string
	IsDefault  bool
}
