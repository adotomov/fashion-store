package domain

import "errors"

var (
	ErrProviderNotFound = errors.New("logistics provider not found")
	ErrProviderDisabled = errors.New("logistics provider is disabled")
)

type ValidationError string

func (e ValidationError) Error() string { return string(e) }
