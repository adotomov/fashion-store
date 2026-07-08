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

// PaymentMethodsFor returns the payment methods that make sense for a given
// delivery method. The two are physically coupled: an EasyBox locker has no
// courier to hand cash to, so it's card-only (paid online up front, or on
// the locker's card terminal at pickup). Courier delivery can be paid online
// up front or on delivery (cash or card to the courier / at a Speedy
// office); only the locker's terminal option doesn't apply to it.
func PaymentMethodsFor(deliveryMethodCode string) []string {
	switch deliveryMethodCode {
	case DeliveryMethodSpeedy:
		return []string{PaymentMethodCashOnDelivery, PaymentMethodCardOnline}
	case DeliveryMethodEasyBox:
		return []string{PaymentMethodCardOnline, PaymentMethodCardOnEasyBox}
	default:
		return nil
	}
}

// PaymentMethodAllowedFor reports whether a payment method is compatible
// with the chosen delivery method (see PaymentMethodsFor).
func PaymentMethodAllowedFor(deliveryMethodCode, paymentMethod string) bool {
	for _, m := range PaymentMethodsFor(deliveryMethodCode) {
		if m == paymentMethod {
			return true
		}
	}
	return false
}

// RequiresUpfrontPayment reports whether the order must be paid before it
// can be placed (mocked Revolut card charge), as opposed to settled in
// person at delivery time.
func RequiresUpfrontPayment(method string) bool {
	return method == PaymentMethodCardOnline
}
