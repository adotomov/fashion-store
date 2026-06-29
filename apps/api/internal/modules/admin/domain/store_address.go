package domain

import (
	"time"

	"github.com/google/uuid"
)

// StoreAddress is one of possibly many physical addresses for a multi-
// location store. Not translated — addresses are always entered in the
// store's working language.
type StoreAddress struct {
	ID              uuid.UUID
	StoreSettingsID uuid.UUID
	Label           string
	Line1           string
	Line2           *string
	City            *string
	Region          *string
	PostalCode      *string
	Country         *string
	IsDefault       bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
