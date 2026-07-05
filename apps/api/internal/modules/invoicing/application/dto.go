package application

import "time"

type ListFilter struct {
	From          *time.Time
	To            *time.Time
	DocumentType  *string
	PaymentMethod *string
	Search        string // matches invoice_number or order_number prefix
	Limit         int
	Offset        int
}
