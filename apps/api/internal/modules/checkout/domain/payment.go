package domain

const (
	PaymentMethodCashOnDelivery = "cash_on_delivery"
	PaymentMethodCardOnEasyBox  = "card_on_easybox"
	PaymentMethodCardOnline     = "card_online"
)

func ValidPaymentMethod(method string) bool {
	switch method {
	case PaymentMethodCashOnDelivery, PaymentMethodCardOnEasyBox, PaymentMethodCardOnline:
		return true
	default:
		return false
	}
}

// RequiresUpfrontPayment reports whether the order must be paid before it
// can be placed (mocked Revolut card charge), as opposed to settled in
// person at delivery time.
func RequiresUpfrontPayment(method string) bool {
	return method == PaymentMethodCardOnline
}
