package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/users/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/users/domain"
	usershttp "github.com/adotomov/fashion-store/apps/api/internal/modules/users/transport/http"
)

type noopRepo struct{}

func (n *noopRepo) FindByID(context.Context, uuid.UUID) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}

func (n *noopRepo) FindByEmail(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}

func (n *noopRepo) Create(context.Context, application.CreateUserInput) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}

func (n *noopRepo) Update(context.Context, domain.User) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}

func (n *noopRepo) ListAddresses(context.Context, uuid.UUID) ([]domain.Address, error) {
	return nil, nil
}

func (n *noopRepo) CreateAddress(context.Context, domain.Address) (*domain.Address, error) {
	return nil, nil
}

func (n *noopRepo) UpdateAddress(context.Context, domain.Address) (*domain.Address, error) {
	return nil, nil
}

func (n *noopRepo) DeleteAddress(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (n *noopRepo) FindAddress(context.Context, uuid.UUID, uuid.UUID) (*domain.Address, error) {
	return nil, domain.ErrAddressNotFound
}

func (n *noopRepo) List(context.Context, application.ListUsersFilter) ([]domain.User, int, error) {
	return nil, 0, nil
}

func (n *noopRepo) SetRoles(context.Context, uuid.UUID, []domain.Role) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}

func (n *noopRepo) Stats(context.Context) (application.UserStats, error) {
	return application.UserStats{}, nil
}

func TestMe_RequiresAuth(t *testing.T) {
	svc := application.NewService(&noopRepo{}, nil)
	handler := usershttp.NewHandler(svc)

	denyAuth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})
	}

	r := chi.NewRouter()
	handler.RegisterRoutes(r, denyAuth, denyAuth)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
