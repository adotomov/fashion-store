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

// PaymentMethodsFor returns the payment methods offered for a given delivery
// method. All three methods — door delivery, office pickup, and EasyBox
// locker — settle either on delivery (Speedy collects the cash-on-delivery
// amount at the door, office, or locker) or online up front. The legacy
// card_on_easybox ("pay on terminal") option is no longer offered, though it
// stays a valid method so historical orders keep rendering.
func PaymentMethodsFor(deliveryMethodCode string) []string {
	switch deliveryMethodCode {
	case DeliveryMethodSpeedy, DeliveryMethodSpeedyOffice, DeliveryMethodEasyBox:
		return []string{PaymentMethodCashOnDelivery, PaymentMethodCardOnline}
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
