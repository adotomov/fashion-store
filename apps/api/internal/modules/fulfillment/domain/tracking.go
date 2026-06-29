package domain

// FriendlyStatus maps a Speedy tracking operation code (see
// .ai/speedy-docs/10-reference.md "Common Tracking Operations") to a short,
// carrier-agnostic status string stored on the order. Unrecognized codes
// fall back to the operation's own description so nothing is silently
// dropped.
func FriendlyStatus(operationCode int, description string) string {
	switch operationCode {
	case 1, 2, 11, 217:
		return "in_transit"
	case 12:
		return "out_for_delivery"
	case 14:
		return "delivered"
	case 39:
		return "picked_up"
	case 44:
		return "exception"
	case 111:
		return "returned"
	case 115:
		return "redirected"
	case 134:
		return "ready_for_pickup"
	case 181:
		return "delayed"
	default:
		return description
	}
}

const StatusDelivered = "delivered"

// IsInFlight reports whether a friendly status means the parcel has left
// our hands but isn't delivered yet — used to bump the order's own status
// to "shipped" the first time tracking shows real carrier movement.
func IsInFlight(status string) bool {
	switch status {
	case "picked_up", "in_transit", "out_for_delivery", "ready_for_pickup":
		return true
	default:
		return false
	}
}
