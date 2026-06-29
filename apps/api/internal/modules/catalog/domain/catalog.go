package domain

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusDraft    Status = "draft"
	StatusActive   Status = "active"
	StatusDisabled Status = "disabled"
)

func (s Status) Valid() bool {
	switch s {
	case StatusDraft, StatusActive, StatusDisabled:
		return true
	default:
		return false
	}
}

type Catalog struct {
	ID          uuid.UUID
	Name        string
	Slug        string
	Description string
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
