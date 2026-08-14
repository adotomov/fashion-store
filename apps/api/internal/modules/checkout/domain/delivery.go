package domain

import "github.com/adotomov/fashion-store/apps/api/internal/shared/money"

// Delivery methods are mocked for now — Speedy and EasyBox with fixed
// fees — until a real logistics integration is wired in. Swapping in real
// carriers later only needs this list (and the admin's tracking fields) to
// change, not the checkout orchestration.
const (
	DeliveryMethodSpeedy       = "speedy"
	DeliveryMethodSpeedyOffice = "speedy_office"
	DeliveryMethodEasyBox      = "easybox"
)

type DeliveryMethod struct {
	Code string
	Name string
	Fee  money.Money
}

func DeliveryMethods() []DeliveryMethod {
	return []DeliveryMethod{
		{Code: DeliveryMethodSpeedy, Name: "Deliver to Address", Fee: money.Money{AmountMinor: 250, Currency: "EUR"}},
		{Code: DeliveryMethodSpeedyOffice, Name: "Deliver to Office", Fee: money.Money{AmountMinor: 250, Currency: "EUR"}},
		{Code: DeliveryMethodEasyBox, Name: "EasyBox Locker", Fee: money.Money{AmountMinor: 0, Currency: "EUR"}},
	}
}

// RequiresOffice reports whether a delivery method is collected from a Speedy
// office or EasyBox locker (so checkout must capture a DeliveryOfficeID) rather
// than delivered to the door.
func RequiresOffice(deliveryMethodCode string) bool {
	switch deliveryMethodCode {
	case DeliveryMethodSpeedyOffice, DeliveryMethodEasyBox:
		return true
	default:
		return false
	}
}

func FindDeliveryMethod(code string) (DeliveryMethod, bool) {
	for _, m := range DeliveryMethods() {
		if m.Code == code {
			return m, true
		}
	}
	return DeliveryMethod{}, false
}

// ProviderFor maps a delivery method to the logistics provider that
// fulfills it. Speedy door delivery, Speedy office pickup and EasyBox lockers
// are all fulfilled through the same Speedy account (differing only by
// destination type — address, OFFICE, or APT), so a single admin toggle
// controls all three.
func ProviderFor(deliveryMethodCode string) string {
	switch deliveryMethodCode {
	case DeliveryMethodSpeedy, DeliveryMethodSpeedyOffice, DeliveryMethodEasyBox:
		return "speedy"
	default:
		return ""
	}
}
