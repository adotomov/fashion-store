package domain

import (
	"time"

	"github.com/google/uuid"
)

// PaymentMethod stores only display metadata for a saved card — brand,
// last 4 digits, and expiry. The full card number and CVV are never
// collected or stored here; a real integration would keep a processor
// token (e.g. Stripe payment_method_id) instead.
type PaymentMethod struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Brand     string
	Last4     string
	ExpMonth  int
	ExpYear   int
	IsDefault bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (p PaymentMethod) Validate() error {
	if p.Brand == "" {
		return ValidationError("brand is required")
	}
	if len(p.Last4) != 4 || !isDigits(p.Last4) {
		return ValidationError("last4 must be exactly 4 digits")
	}
	if p.ExpMonth < 1 || p.ExpMonth > 12 {
		return ValidationError("exp_month must be between 1 and 12")
	}
	if p.ExpYear < 2000 || p.ExpYear > 2100 {
		return ValidationError("exp_year is invalid")
	}
	return nil
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
