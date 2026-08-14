package domain

import "testing"

func TestPaymentMethodAllowedFor(t *testing.T) {
	cases := []struct {
		delivery string
		payment  string
		want     bool
	}{
		// Courier can be paid on delivery or online; the locker terminal doesn't apply.
		{DeliveryMethodSpeedy, PaymentMethodCashOnDelivery, true},
		{DeliveryMethodSpeedy, PaymentMethodCardOnline, true},
		{DeliveryMethodSpeedy, PaymentMethodCardOnEasyBox, false},
		// Office pickup behaves like courier: cash at the office or online.
		{DeliveryMethodSpeedyOffice, PaymentMethodCashOnDelivery, true},
		{DeliveryMethodSpeedyOffice, PaymentMethodCardOnline, true},
		{DeliveryMethodSpeedyOffice, PaymentMethodCardOnEasyBox, false},
		// A locker settles on delivery (Speedy COD at the locker) or online;
		// the legacy terminal option is no longer offered.
		{DeliveryMethodEasyBox, PaymentMethodCashOnDelivery, true},
		{DeliveryMethodEasyBox, PaymentMethodCardOnline, true},
		{DeliveryMethodEasyBox, PaymentMethodCardOnEasyBox, false},
		// Unknown delivery method allows nothing.
		{"pigeon", PaymentMethodCashOnDelivery, false},
	}
	for _, c := range cases {
		if got := PaymentMethodAllowedFor(c.delivery, c.payment); got != c.want {
			t.Errorf("PaymentMethodAllowedFor(%q, %q) = %v, want %v", c.delivery, c.payment, got, c.want)
		}
	}
}

func TestPaymentMethodsForOnlyReturnsValidMethods(t *testing.T) {
	for _, delivery := range []string{DeliveryMethodSpeedy, DeliveryMethodSpeedyOffice, DeliveryMethodEasyBox} {
		methods := PaymentMethodsFor(delivery)
		if len(methods) == 0 {
			t.Errorf("PaymentMethodsFor(%q) returned no methods", delivery)
		}
		for _, m := range methods {
			if !ValidPaymentMethod(m) {
				t.Errorf("PaymentMethodsFor(%q) returned invalid method %q", delivery, m)
			}
		}
	}
}
