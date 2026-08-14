package domain

import (
	"time"

	"github.com/google/uuid"
)

// Address is a Speedy-resolved delivery address: the city, neighbourhood
// (complex/кв./жк.) and street are carried as Speedy location codes (SiteID,
// ComplexID, StreetID) alongside their display names, so a shipment can be
// created without the carrier having to guess-resolve typed text. Bulgaria is
// the only supported country for now (CountryCode "BG", CountryID 100).
type Address struct {
	ID            uuid.UUID
	UserID        uuid.UUID
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
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (a Address) Validate() error {
	if a.RecipientName == "" {
		return ErrAddressInvalid("recipient_name is required")
	}
	if a.SiteID <= 0 {
		return ErrAddressInvalid("city (site) is required")
	}
	if a.ComplexID <= 0 {
		return ErrAddressInvalid("complex (кв./жк.) is required")
	}
	if a.StreetID <= 0 {
		return ErrAddressInvalid("street is required")
	}
	if len(a.CountryCode) != 2 {
		return ErrAddressInvalid("country_code must be a 2-letter ISO code")
	}
	return nil
}
