package domain

import (
	"time"

	"github.com/google/uuid"
)

type Address struct {
	ID            uuid.UUID
	UserID        uuid.UUID
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
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (a Address) Validate() error {
	if a.RecipientName == "" {
		return ErrAddressInvalid("recipient_name is required")
	}
	if a.Line1 == "" {
		return ErrAddressInvalid("line1 is required")
	}
	if a.City == "" {
		return ErrAddressInvalid("city is required")
	}
	if a.PostalCode == "" {
		return ErrAddressInvalid("postal_code is required")
	}
	if len(a.CountryCode) != 2 {
		return ErrAddressInvalid("country_code must be a 2-letter ISO code")
	}
	return nil
}
