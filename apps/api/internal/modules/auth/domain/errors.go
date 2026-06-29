package domain

import "errors"

var (
	ErrIdentityNotFound = errors.New("identity not found")
	ErrSessionNotFound  = errors.New("session not found")
	ErrSessionExpired   = errors.New("session expired or revoked")
	ErrInvalidToken     = errors.New("invalid bearer token")
)
