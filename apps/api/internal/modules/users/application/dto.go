package application

type UpdateProfileInput struct {
	FullName *string
	Phone    *string
}

type AddAddressInput struct {
	Label         string
	RecipientName string
	Phone         string
	Line1         string
	Line2         string
	City          string
	Region        string
	PostalCode    string
	CountryCode   string
	IsDefault     bool
}

type UpdateAddressInput struct {
	Label         *string
	RecipientName *string
	Phone         *string
	Line1         *string
	Line2         *string
	City          *string
	Region        *string
	PostalCode    *string
	CountryCode   *string
	IsDefault     *bool
}
