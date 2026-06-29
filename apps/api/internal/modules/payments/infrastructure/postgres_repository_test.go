package infrastructure_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/payments/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/payments/infrastructure"
	usersapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/users/application"
	usersinfra "github.com/adotomov/fashion-store/apps/api/internal/modules/users/infrastructure"
)

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping Postgres integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestPostgresRepository_DefaultExclusivity(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	usersRepo := usersinfra.NewPostgresRepository(pool)
	user, err := usersRepo.FindByEmail(ctx, "it-test-payments@example.com")
	if err != nil {
		user, err = usersRepo.Create(ctx, usersapplication.CreateUserInput{
			Email: "it-test-payments@example.com", FullName: "IT Test Payments",
		})
		if err != nil {
			t.Fatalf("create user: %v", err)
		}
	}

	repo := infrastructure.NewPostgresRepository(pool)
	if _, err := pool.Exec(ctx, `DELETE FROM payment_methods WHERE user_id = $1`, user.ID); err != nil {
		t.Fatalf("cleanup prior test methods: %v", err)
	}

	first, err := repo.Create(ctx, domain.PaymentMethod{
		UserID: user.ID, Brand: "visa", Last4: "4242", ExpMonth: 12, ExpYear: 2030, IsDefault: true,
	})
	if err != nil {
		t.Fatalf("create first method: %v", err)
	}
	if !first.IsDefault {
		t.Fatalf("expected first method to be default")
	}

	second, err := repo.Create(ctx, domain.PaymentMethod{
		UserID: user.ID, Brand: "mastercard", Last4: "5555", ExpMonth: 6, ExpYear: 2031, IsDefault: true,
	})
	if err != nil {
		t.Fatalf("create second method: %v", err)
	}
	if !second.IsDefault {
		t.Fatalf("expected second method to be default")
	}

	refreshedFirst, err := repo.Find(ctx, user.ID, first.ID)
	if err != nil {
		t.Fatalf("find first method: %v", err)
	}
	if refreshedFirst.IsDefault {
		t.Errorf("expected first method to no longer be default after second was created as default")
	}

	methods, err := repo.ListByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("list methods: %v", err)
	}
	defaultCount := 0
	for _, m := range methods {
		if m.IsDefault {
			defaultCount++
		}
	}
	if defaultCount != 1 {
		t.Errorf("expected exactly 1 default payment method, got %d", defaultCount)
	}

	if err := repo.Delete(ctx, user.ID, first.ID); err != nil {
		t.Fatalf("delete first method: %v", err)
	}
	if err := repo.Delete(ctx, user.ID, second.ID); err != nil {
		t.Fatalf("delete second method: %v", err)
	}
}
