package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleUser        Role = "user"
	RoleAdmin       Role = "admin"
	RoleAudit       Role = "audit"
	RoleAccountant  Role = "accountant"
)

type User struct {
	ID        uuid.UUID
	Email     string
	FullName  string
	Phone     string
	Roles     []Role
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (u User) HasRole(role Role) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}
