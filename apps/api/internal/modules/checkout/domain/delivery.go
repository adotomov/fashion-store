package domain

import "github.com/adotomov/fashion-store/apps/api/internal/shared/money"

// Delivery methods are mocked for now — Speedy and EasyBox with fixed
// fees — until a real logistics integration is wired in. Swapping in real
// carriers later only needs this list (and the admin's tracking fields) to
// change, not the checkout orchestration.
const (
	DeliveryMethodSpeedy  = "speedy"
	DeliveryMethodEasyBox = "easybox"
)

type DeliveryMethod struct {
	Code string
	Name string
	Fee  money.Money
}

func DeliveryMethods() []DeliveryMethod {
	return []DeliveryMethod{
		{Code: DeliveryMethodSpeedy, Name: "Speedy Courier", Fee: money.Money{AmountMinor: 299, Currency: "EUR"}},
		{Code: DeliveryMethodEasyBox, Name: "EasyBox Locker", Fee: money.Money{AmountMinor: 0, Currency: "EUR"}},
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
// fulfills it. Both speedy and easybox are fulfilled through the same
// Speedy account (door delivery vs. APT/locker office type), so a single
// admin toggle controls both.
func ProviderFor(deliveryMethodCode string) string {
	switch deliveryMethodCode {
	case DeliveryMethodSpeedy, DeliveryMethodEasyBox:
		return "speedy"
	default:
		return ""
	}
}
