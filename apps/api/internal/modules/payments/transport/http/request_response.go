package http

import (
	"github.com/adotomov/fashion-store/apps/api/internal/modules/payments/domain"
)

type paymentMethodResponse struct {
	ID        string `json:"id"`
	Brand     string `json:"brand"`
	Last4     string `json:"last4"`
	ExpMonth  int    `json:"exp_month"`
	ExpYear   int    `json:"exp_year"`
	IsDefault bool   `json:"is_default"`
}

func toPaymentMethodResponse(m domain.PaymentMethod) paymentMethodResponse {
	return paymentMethodResponse{
		ID:        m.ID.String(),
		Brand:     m.Brand,
		Last4:     m.Last4,
		ExpMonth:  m.ExpMonth,
		ExpYear:   m.ExpYear,
		IsDefault: m.IsDefault,
	}
}
