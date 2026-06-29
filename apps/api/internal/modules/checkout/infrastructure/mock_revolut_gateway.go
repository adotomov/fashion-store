package infrastructure

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/checkout/application"
)

// MockRevolutGateway emulates the outcome shape of the Revolut Merchant
// API (an order reference plus a succeeded/failed result) without calling
// out to Revolut — the merchant account isn't verified yet. Swap this for
// a real client implementing application.PaymentGateway once it is; the
// checkout service never needs to change.
type MockRevolutGateway struct{}

func NewMockRevolutGateway() *MockRevolutGateway {
	return &MockRevolutGateway{}
}

func (g *MockRevolutGateway) Charge(_ context.Context, input application.ChargeInput) (application.ChargeResult, error) {
	// Deterministic test hook: a card number ending in "0000" simulates a
	// decline, so the reservation-release-on-failure path can be exercised
	// without a real payment failure.
	if strings.HasSuffix(input.Card.Number, "0000") {
		return application.ChargeResult{Succeeded: false, FailureReason: "card_declined"}, nil
	}
	return application.ChargeResult{
		Succeeded:         true,
		ProviderReference: "mock_rev_" + uuid.NewString(),
	}, nil
}
