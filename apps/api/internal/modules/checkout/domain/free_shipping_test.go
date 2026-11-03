package domain

import "testing"

func TestDeliveryFeeWaivedAtAndAboveThreshold(t *testing.T) {
	speedy, _ := FindDeliveryMethod(DeliveryMethodSpeedy)
	const threshold = 5100

	if got := DeliveryFee(speedy, threshold-1, threshold); got.AmountMinor != 250 {
		t.Fatalf("below the threshold the fee is the method's own: got %d, want 250", got.AmountMinor)
	}
	if got := DeliveryFee(speedy, threshold, threshold); got.AmountMinor != 0 || got.Currency != "EUR" {
		t.Fatalf("at the threshold delivery is free: got %+v", got)
	}
	if got := DeliveryFee(speedy, threshold+1000, threshold); got.AmountMinor != 0 {
		t.Fatalf("above the threshold delivery is free: got %+v", got)
	}
}

func TestDeliveryFeeOfAFreeMethodIsUnchanged(t *testing.T) {
	easybox, _ := FindDeliveryMethod(DeliveryMethodEasyBox)
	if got := DeliveryFee(easybox, 0, 5100); got.AmountMinor != 0 || got.Currency != "EUR" {
		t.Fatalf("EasyBox is free either way: got %+v", got)
	}
}
