package domain

import "errors"

var (
	ErrInvoiceNotFound       = errors.New("invoice not found")
	ErrInvoiceAlreadyExists  = errors.New("an invoice already exists for this order")
	ErrSettingsIncomplete    = errors.New("invoice settings are incomplete: company EIK and NRA store number must be configured")
	ErrCannotStornoAStorno   = errors.New("cannot issue a storno on a storno document")
)
