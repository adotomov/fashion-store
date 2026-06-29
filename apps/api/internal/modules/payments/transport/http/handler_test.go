package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/payments/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/payments/infrastructure"
	paymentshttp "github.com/adotomov/fashion-store/apps/api/internal/modules/payments/transport/http"
	usersapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/users/application"
	usersinfra "github.com/adotomov/fashion-store/apps/api/internal/modules/users/infrastructure"

	"github.com/adotomov/fashion-store/apps/api/internal/shared/authctx"
)

// injectPrincipal stands in for the real auth middleware so these tests
// exercise routing/JSON/repo behavior without a live session.
func injectPrincipal(userID uuid.UUID) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := authctx.WithPrincipal(r.Context(), authctx.Principal{UserID: userID})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func TestPaymentMethodsHTTP_CreateValidationAndDefault(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping Postgres integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	t.Cleanup(pool.Close)

	ctx := context.Background()
	usersRepo := usersinfra.NewPostgresRepository(pool)
	user, err := usersRepo.FindByEmail(ctx, "it-test-payments-http@example.com")
	if err != nil {
		user, err = usersRepo.Create(ctx, usersapplication.CreateUserInput{
			Email: "it-test-payments-http@example.com", FullName: "IT Test Payments HTTP",
		})
		if err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	if _, err := pool.Exec(ctx, `DELETE FROM payment_methods WHERE user_id = $1`, user.ID); err != nil {
		t.Fatalf("cleanup prior test methods: %v", err)
	}

	repo := infrastructure.NewPostgresRepository(pool)
	service := application.NewService(repo)
	handler := paymentshttp.NewHandler(service)

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		handler.RegisterRoutes(r, injectPrincipal(user.ID))
	})

	// Invalid last4 (3 digits) should be rejected with 400.
	badBody := `{"brand":"visa","last4":"123","exp_month":12,"exp_year":2030}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/payment-methods", strings.NewReader(badBody))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid last4, got %d: %s", rec.Code, rec.Body.String())
	}

	// Valid creation as default.
	goodBody := `{"brand":"visa","last4":"4242","exp_month":12,"exp_year":2030,"is_default":true}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/me/payment-methods", strings.NewReader(goodBody))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/me/payment-methods", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var methods []struct {
		Last4     string `json:"last4"`
		IsDefault bool   `json:"is_default"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &methods); err != nil {
		t.Fatalf("decode methods: %v", err)
	}
	if len(methods) != 1 || methods[0].Last4 != "4242" || !methods[0].IsDefault {
		t.Fatalf("expected 1 default method with last4=4242, got %+v", methods)
	}
}
