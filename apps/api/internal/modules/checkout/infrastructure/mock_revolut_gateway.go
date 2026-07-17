package infrastructure

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/checkout/application"
)

// MockRevolutGateway emulates the Revolut Merchant order lifecycle without
// calling out to Revolut — used locally/in devbox when no REVOLUT_API_KEY is
// configured. It keeps created orders in memory so a simulated webhook can
// drive FinalizePaidOrder end-to-end: GetOrder echoes the stored amount with a
// "completed" state. Swap for RevolutGateway once the merchant account is live;
// the checkout service never changes.
type MockRevolutGateway struct {
	mu     sync.Mutex
	orders map[string]application.PaymentOrder
}

func NewMockRevolutGateway() *MockRevolutGateway {
	return &MockRevolutGateway{orders: make(map[string]application.PaymentOrder)}
}

func (g *MockRevolutGateway) CreateOrder(_ context.Context, input application.CreatePaymentOrderInput) (application.PaymentOrder, error) {
	id := "mock_rev_" + uuid.NewString()
	order := application.PaymentOrder{
		ID:          id,
		Token:       "mock_token_" + id,
		State:       "pending",
		AmountMinor: input.Amount.AmountMinor,
		Currency:    input.Amount.Currency,
	}
	g.mu.Lock()
	g.orders[id] = order
	g.mu.Unlock()
	return order, nil
}

func (g *MockRevolutGateway) GetOrder(_ context.Context, providerOrderID string) (application.PaymentOrder, error) {
	g.mu.Lock()
	order, ok := g.orders[providerOrderID]
	g.mu.Unlock()
	if !ok {
		return application.PaymentOrder{ID: providerOrderID, State: application.PaymentStateCompleted}, nil
	}
	order.State = application.PaymentStateCompleted
	return order, nil
}

func (g *MockRevolutGateway) Refund(_ context.Context, input application.RefundInput) (application.RefundResult, error) {
	return application.RefundResult{ID: "mock_refund_" + uuid.NewString(), State: "completed"}, nil
}
