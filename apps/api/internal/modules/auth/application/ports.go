package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/auth/domain"
)

// GoogleVerifier verifies a Google ID token and returns the identity it
// asserts. Implemented in internal/platform/googleauth to keep the Google
// SDK out of application/domain logic.
type GoogleVerifier interface {
	Verify(ctx context.Context, idToken string) (GoogleIdentity, error)
}

type GoogleIdentity struct {
	Subject       string
	Email         string
	EmailVerified bool
	FullName      string
}

// IdentityRepository persists external auth identities and links them to
// local users.
type IdentityRepository interface {
	FindByProviderSubject(ctx context.Context, provider, subject string) (*domain.Identity, error)
	Create(ctx context.Context, identity domain.Identity) (*domain.Identity, error)
}

// SessionRepository persists backend-controlled sessions.
type SessionRepository interface {
	Create(ctx context.Context, session domain.Session) (*domain.Session, error)
	FindByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error)
	Revoke(ctx context.Context, sessionID uuid.UUID) error
}

// UserProvisioner creates or finds the local user behind an authenticated
// identity. Implemented by an adapter over the users module's application
// service so auth never imports users' infrastructure.
type UserProvisioner interface {
	EnsureUser(ctx context.Context, input EnsureUserInput) (UserRef, error)
	GetRoles(ctx context.Context, userID uuid.UUID) ([]string, error)
}

type EnsureUserInput struct {
	Email    string
	FullName string
}

type UserRef struct {
	ID uuid.UUID
}
