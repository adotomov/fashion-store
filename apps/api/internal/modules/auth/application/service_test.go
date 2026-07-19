package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/auth/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/auth/domain"
)

type fakeVerifier struct {
	identity application.GoogleIdentity
	err      error
}

func (f *fakeVerifier) Verify(_ context.Context, _ string) (application.GoogleIdentity, error) {
	return f.identity, f.err
}

type fakeIdentityRepo struct {
	byProviderSubject map[string]domain.Identity
}

func newFakeIdentityRepo() *fakeIdentityRepo {
	return &fakeIdentityRepo{byProviderSubject: map[string]domain.Identity{}}
}

func (f *fakeIdentityRepo) FindByProviderSubject(_ context.Context, provider, subject string) (*domain.Identity, error) {
	id, ok := f.byProviderSubject[provider+":"+subject]
	if !ok {
		return nil, domain.ErrIdentityNotFound
	}
	return &id, nil
}

func (f *fakeIdentityRepo) Create(_ context.Context, identity domain.Identity) (*domain.Identity, error) {
	identity.ID = uuid.New()
	f.byProviderSubject[identity.Provider+":"+identity.ProviderSubject] = identity
	return &identity, nil
}

type fakeSessionRepo struct {
	byHash map[string]domain.Session
}

func newFakeSessionRepo() *fakeSessionRepo {
	return &fakeSessionRepo{byHash: map[string]domain.Session{}}
}

func (f *fakeSessionRepo) Create(_ context.Context, session domain.Session) (*domain.Session, error) {
	session.ID = uuid.New()
	session.CreatedAt = time.Now()
	f.byHash[session.TokenHash] = session
	return &session, nil
}

func (f *fakeSessionRepo) FindByTokenHash(_ context.Context, tokenHash string) (*domain.Session, error) {
	s, ok := f.byHash[tokenHash]
	if !ok {
		return nil, domain.ErrSessionNotFound
	}
	return &s, nil
}

func (f *fakeSessionRepo) Revoke(_ context.Context, sessionID uuid.UUID) error {
	for k, s := range f.byHash {
		if s.ID == sessionID {
			now := time.Now()
			s.RevokedAt = &now
			f.byHash[k] = s
		}
	}
	return nil
}

type fakeProvisioner struct {
	usersByEmail map[string]uuid.UUID
	roles        map[uuid.UUID][]string
}

func newFakeProvisioner() *fakeProvisioner {
	return &fakeProvisioner{usersByEmail: map[string]uuid.UUID{}, roles: map[uuid.UUID][]string{}}
}

func (f *fakeProvisioner) EnsureUser(_ context.Context, input application.EnsureUserInput) (application.UserRef, error) {
	if id, ok := f.usersByEmail[input.Email]; ok {
		return application.UserRef{ID: id}, nil
	}
	id := uuid.New()
	f.usersByEmail[input.Email] = id
	f.roles[id] = []string{"user"}
	return application.UserRef{ID: id}, nil
}

func (f *fakeProvisioner) GetRoles(_ context.Context, userID uuid.UUID) ([]string, error) {
	return f.roles[userID], nil
}

func TestLoginWithGoogle_CreatesUserAndIdentityOnFirstLogin(t *testing.T) {
	verifier := &fakeVerifier{identity: application.GoogleIdentity{
		Subject: "google-subject-1", Email: "jane@example.com", FullName: "Jane Doe", EmailVerified: true,
	}}
	identities := newFakeIdentityRepo()
	sessions := newFakeSessionRepo()
	provisioner := newFakeProvisioner()

	svc := application.NewService(verifier, identities, sessions, provisioner, time.Hour)

	result, err := svc.LoginWithGoogle(context.Background(), "fake-id-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Token == "" {
		t.Fatal("expected a non-empty token")
	}
	if !result.IsNew {
		t.Fatal("expected IsNew=true on first registration")
	}

	if _, err := identities.FindByProviderSubject(context.Background(), "google", "google-subject-1"); err != nil {
		t.Fatalf("expected identity to be created, got error: %v", err)
	}
}

func TestLoginWithGoogle_ReusesUserOnRepeatLogin(t *testing.T) {
	verifier := &fakeVerifier{identity: application.GoogleIdentity{
		Subject: "google-subject-2", Email: "john@example.com", FullName: "John Doe", EmailVerified: true,
	}}
	identities := newFakeIdentityRepo()
	sessions := newFakeSessionRepo()
	provisioner := newFakeProvisioner()

	svc := application.NewService(verifier, identities, sessions, provisioner, time.Hour)

	first, err := svc.LoginWithGoogle(context.Background(), "fake-id-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	second, err := svc.LoginWithGoogle(context.Background(), "fake-id-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if first.UserID != second.UserID {
		t.Fatalf("expected same user id across logins, got %v and %v", first.UserID, second.UserID)
	}
	if !first.IsNew {
		t.Fatal("expected IsNew=true on the first login")
	}
	if second.IsNew {
		t.Fatal("expected IsNew=false on a repeat login")
	}
	if len(provisioner.usersByEmail) != 1 {
		t.Fatalf("expected exactly one provisioned user, got %d", len(provisioner.usersByEmail))
	}
}

func TestAuthenticate_FailsForExpiredSession(t *testing.T) {
	verifier := &fakeVerifier{identity: application.GoogleIdentity{
		Subject: "google-subject-3", Email: "expired@example.com", FullName: "Expired User", EmailVerified: true,
	}}
	identities := newFakeIdentityRepo()
	sessions := newFakeSessionRepo()
	provisioner := newFakeProvisioner()

	svc := application.NewService(verifier, identities, sessions, provisioner, -time.Hour)

	loginResult, err := svc.LoginWithGoogle(context.Background(), "fake-id-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := svc.Authenticate(context.Background(), loginResult.Token); err == nil {
		t.Fatal("expected expired session to fail authentication")
	}
}

func TestAuthenticate_FailsForUnknownToken(t *testing.T) {
	identities := newFakeIdentityRepo()
	sessions := newFakeSessionRepo()
	provisioner := newFakeProvisioner()
	verifier := &fakeVerifier{}

	svc := application.NewService(verifier, identities, sessions, provisioner, time.Hour)

	if _, err := svc.Authenticate(context.Background(), "does-not-exist"); err == nil {
		t.Fatal("expected unknown token to fail authentication")
	}
}
