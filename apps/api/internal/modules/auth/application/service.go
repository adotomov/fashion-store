package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/auth/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/authctx"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/tokens"
)

const googleProvider = "google"

type Service struct {
	verifier    GoogleVerifier
	identities  IdentityRepository
	sessions    SessionRepository
	provisioner UserProvisioner
	sessionTTL  time.Duration
}

func NewService(verifier GoogleVerifier, identities IdentityRepository, sessions SessionRepository, provisioner UserProvisioner, sessionTTL time.Duration) *Service {
	return &Service{
		verifier:    verifier,
		identities:  identities,
		sessions:    sessions,
		provisioner: provisioner,
		sessionTTL:  sessionTTL,
	}
}

type LoginResult struct {
	Token     string
	ExpiresAt time.Time
	UserID    uuid.UUID
}

// LoginWithGoogle verifies the Google ID token, provisions or links the
// local user, and issues a new backend session/bearer token.
func (s *Service) LoginWithGoogle(ctx context.Context, idToken string) (*LoginResult, error) {
	identity, err := s.verifier.Verify(ctx, idToken)
	if err != nil {
		return nil, err
	}

	existing, err := s.identities.FindByProviderSubject(ctx, googleProvider, identity.Subject)
	var userID uuid.UUID
	if err == nil {
		userID = existing.UserID
	} else if errors.Is(err, domain.ErrIdentityNotFound) {
		user, err := s.provisioner.EnsureUser(ctx, EnsureUserInput{
			Email:    identity.Email,
			FullName: identity.FullName,
		})
		if err != nil {
			return nil, err
		}
		userID = user.ID

		if _, err := s.identities.Create(ctx, domain.Identity{
			UserID:          userID,
			Provider:        googleProvider,
			ProviderSubject: identity.Subject,
			Email:           identity.Email,
			EmailVerified:   identity.EmailVerified,
		}); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}

	return s.issueSession(ctx, userID)
}

func (s *Service) issueSession(ctx context.Context, userID uuid.UUID) (*LoginResult, error) {
	token, hash := tokens.Generate()
	expiresAt := time.Now().Add(s.sessionTTL)

	if _, err := s.sessions.Create(ctx, domain.Session{
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: expiresAt,
	}); err != nil {
		return nil, err
	}

	return &LoginResult{Token: token, ExpiresAt: expiresAt, UserID: userID}, nil
}

// Refresh issues a new session for the user behind a still-valid bearer
// token, then revokes the old session.
func (s *Service) Refresh(ctx context.Context, token string) (*LoginResult, error) {
	session, err := s.lookupValidSession(ctx, token)
	if err != nil {
		return nil, err
	}

	result, err := s.issueSession(ctx, session.UserID)
	if err != nil {
		return nil, err
	}

	_ = s.sessions.Revoke(ctx, session.ID)

	return result, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	session, err := s.lookupValidSession(ctx, token)
	if err != nil {
		return err
	}
	return s.sessions.Revoke(ctx, session.ID)
}

// Authenticate validates a bearer token and returns the principal (user id
// + server-evaluated roles) for downstream authorization checks.
func (s *Service) Authenticate(ctx context.Context, token string) (authctx.Principal, error) {
	session, err := s.lookupValidSession(ctx, token)
	if err != nil {
		return authctx.Principal{}, err
	}

	roles, err := s.provisioner.GetRoles(ctx, session.UserID)
	if err != nil {
		return authctx.Principal{}, err
	}

	return authctx.Principal{UserID: session.UserID, Roles: roles}, nil
}

func (s *Service) lookupValidSession(ctx context.Context, token string) (*domain.Session, error) {
	hash := tokens.Hash(token)
	session, err := s.sessions.FindByTokenHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if !session.IsValid(time.Now()) {
		return nil, domain.ErrSessionExpired
	}
	return session, nil
}
