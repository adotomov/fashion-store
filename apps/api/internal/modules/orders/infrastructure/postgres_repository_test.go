package infrastructure_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	ordersapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/orders/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/orders/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/orders/infrastructure"
	usersapplication "github.com/adotomov/fashion-store/apps/api/internal/modules/users/application"
	usersinfra "github.com/adotomov/fashion-store/apps/api/internal/modules/users/infrastructure"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

// Integration test against a real Postgres instance. Skips automatically if
// DATABASE_URL isn't set, per 23-testing-guidelines.md.
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

func TestPostgresRepository_CreateAndListOrders(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	usersRepo := usersinfra.NewPostgresRepository(pool)
	user, err := usersRepo.FindByEmail(ctx, "it-test-orders@example.com")
	if err != nil {
		user, err = usersRepo.Create(ctx, usersapplication.CreateUserInput{
			Email: "it-test-orders@example.com", FullName: "IT Test Orders",
		})
		if err != nil {
			t.Fatalf("create user: %v", err)
		}
	}

	orderRepo := infrastructure.NewPostgresRepository(pool)

	// Re-running this test against a persistent dev database would otherwise
	// collide with the unique order_number constraint from a prior run.
	if _, err := pool.Exec(ctx, `DELETE FROM orders WHERE order_number = 'IT-TEST-0001'`); err != nil {
		t.Fatalf("cleanup prior test order: %v", err)
	}

	order, err := orderRepo.Create(ctx, domain.Order{
		UserID:      user.ID,
		OrderNumber: "IT-TEST-0001",
		Status:      domain.OrderStatusPaid,
		Total:       money.Money{AmountMinor: 6000, Currency: "EUR"},
		PlacedAt:    time.Now(),
		Items: []domain.OrderItem{
			{ProductName: "Silk Warp Dress", VariantLabel: "Size: M", Quantity: 1, UnitPrice: money.Money{AmountMinor: 3500, Currency: "EUR"}},
			{ProductName: "Bracelet", VariantLabel: "Size: S", Quantity: 1, UnitPrice: money.Money{AmountMinor: 2500, Currency: "EUR"}},
		},
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if len(order.Items) != 2 {
		t.Fatalf("expected 2 items on created order, got %d", len(order.Items))
	}

	orders, err := orderRepo.ListByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("list orders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(orders))
	}
	if orders[0].OrderNumber != "IT-TEST-0001" {
		t.Errorf("expected order number IT-TEST-0001, got %s", orders[0].OrderNumber)
	}
	if len(orders[0].Items) != 2 {
		t.Errorf("expected 2 items when listing, got %d", len(orders[0].Items))
	}
	if orders[0].Total.AmountMinor != 6000 {
		t.Errorf("expected total 6000, got %d", orders[0].Total.AmountMinor)
	}
}

// TestPostgresRepository_PaymentLedger exercises the append-only audit ledger
// across a card order's lifecycle: initiated → captured → refunded, plus the
// idempotency guarantee that a duplicate MarkPaid appends no second row.
func TestPostgresRepository_PaymentLedger(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	usersRepo := usersinfra.NewPostgresRepository(pool)
	user, err := usersRepo.FindByEmail(ctx, "it-test-ledger@example.com")
	if err != nil {
		user, err = usersRepo.Create(ctx, usersapplication.CreateUserInput{
			Email: "it-test-ledger@example.com", FullName: "IT Test Ledger",
		})
		if err != nil {
			t.Fatalf("create user: %v", err)
		}
	}

	orderRepo := infrastructure.NewPostgresRepository(pool)

	// Cascades to order_payments / order_refunds / payment_transactions.
	if _, err := pool.Exec(ctx, `DELETE FROM orders WHERE order_number = 'IT-TEST-LEDGER-0001'`); err != nil {
		t.Fatalf("cleanup prior test order: %v", err)
	}

	const providerOrderID = "rev_it_ledger_0001"
	order, err := orderRepo.Create(ctx, domain.Order{
		UserID:      user.ID,
		OrderNumber: "IT-TEST-LEDGER-0001",
		Status:      domain.OrderStatusPendingPayment,
		Total:       money.Money{AmountMinor: 5000, Currency: "EUR"},
		PlacedAt:    time.Now(),
		Payment: &domain.OrderPayment{
			Provider:        "revolut",
			ProviderOrderID: providerOrderID,
			Status:          "pending",
			Amount:          money.Money{AmountMinor: 5000, Currency: "EUR"},
		},
		Items: []domain.OrderItem{
			{ProductName: "Silk Dress", Quantity: 1, UnitPrice: money.Money{AmountMinor: 5000, Currency: "EUR"}},
		},
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	// Creating a card order records the 'initiated' entry.
	assertLedgerCounts(t, orderRepo, order.ID, map[string]int{domain.PaymentTxnInitiated: 1})

	// Settle it — appends 'captured'. Calling twice must stay idempotent.
	if err := orderRepo.MarkPaid(ctx, order.ID, "rev_payment_0001", 5000); err != nil {
		t.Fatalf("mark paid: %v", err)
	}
	if err := orderRepo.MarkPaid(ctx, order.ID, "rev_payment_0001", 5000); err != nil {
		t.Fatalf("mark paid (repeat): %v", err)
	}
	assertLedgerCounts(t, orderRepo, order.ID, map[string]int{
		domain.PaymentTxnInitiated: 1,
		domain.PaymentTxnCaptured:  1,
	})

	// Full refund — appends 'refunded' and rolls the order to refunded.
	if err := orderRepo.RecordRefund(ctx, ordersapplication.RecordRefundInput{
		OrderID:          order.ID,
		ProviderRefundID: "rev_refund_0001",
		AmountMinor:      5000,
		Currency:         "EUR",
		Reason:           "customer request",
		State:            "completed",
		OrderStatus:      string(domain.OrderStatusRefunded),
	}); err != nil {
		t.Fatalf("record refund: %v", err)
	}
	assertLedgerCounts(t, orderRepo, order.ID, map[string]int{
		domain.PaymentTxnInitiated: 1,
		domain.PaymentTxnCaptured:  1,
		domain.PaymentTxnRefunded:  1,
	})

	txns, err := orderRepo.ListPaymentTransactions(ctx, order.ID)
	if err != nil {
		t.Fatalf("list payment transactions: %v", err)
	}
	if len(txns) != 3 {
		t.Fatalf("expected 3 ledger rows, got %d", len(txns))
	}
	for _, txn := range txns {
		if txn.Provider != "revolut" || txn.ProviderOrderID != providerOrderID {
			t.Errorf("ledger row %s not tied back to the payment: provider=%q order=%q", txn.Type, txn.Provider, txn.ProviderOrderID)
		}
	}
}

func assertLedgerCounts(t *testing.T, repo *infrastructure.PostgresRepository, orderID uuid.UUID, want map[string]int) {
	t.Helper()
	txns, err := repo.ListPaymentTransactions(context.Background(), orderID)
	if err != nil {
		t.Fatalf("list payment transactions: %v", err)
	}
	got := map[string]int{}
	for _, txn := range txns {
		got[txn.Type]++
	}
	for typ, n := range want {
		if got[typ] != n {
			t.Errorf("expected %d %q ledger rows, got %d", n, typ, got[typ])
		}
	}
}
