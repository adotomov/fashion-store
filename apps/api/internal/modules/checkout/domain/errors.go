package domain

import "errors"

var (
	ErrCartEmpty                 = errors.New("cart is empty")
	ErrInvalidDeliveryMethod     = errors.New("invalid delivery method")
	ErrInvalidPaymentMethod      = errors.New("invalid payment method")
	ErrPaymentMethodNotAllowed   = errors.New("payment method is not available for the chosen delivery method")
	ErrInsufficientStock         = errors.New("not enough stock available")
	ErrPaymentFailed             = errors.New("payment failed")
	ErrDeliveryMethodUnavailable = errors.New("delivery method is currently unavailable")
	ErrOfficeRequired            = errors.New("a pickup locker is required for this delivery method")
	ErrInvalidDiscountCode       = errors.New("discount code is invalid, expired, or exhausted")

	// ErrPaymentInitiation is returned when opening the Revolut order for an
	// online-card checkout fails before the customer can pay.
	ErrPaymentInitiation = errors.New("could not start card payment")

	// Settlement / refund errors (used by the webhook + admin refund paths).
	ErrOrderNotFound         = errors.New("order not found")
	ErrPaymentAmountMismatch = errors.New("paid amount does not match the order total")
	ErrRefundNotAllowed      = errors.New("order cannot be refunded")
	ErrRefundAmountInvalid   = errors.New("refund amount is invalid")
	ErrRefundFailed          = errors.New("refund could not be processed")

	// Shipment retry errors (admin "retry shipment booking" action).
	// ErrShipmentNotRetryable is returned for an order whose delivery method has
	// no logistics provider (nothing to book).
	ErrShipmentNotRetryable = errors.New("order has no shippable delivery method")
	// ErrShipmentAlreadyBooked guards against creating a duplicate label when a
	// Speedy shipment was already booked for the order.
	ErrShipmentAlreadyBooked = errors.New("a shipment has already been booked for this order")
	// ErrShipmentBookingFailed wraps a failed carrier booking on retry; the
	// underlying reason is also recorded on the order for the admin to see.
	ErrShipmentBookingFailed = errors.New("shipment booking failed")
)

type ValidationError string

func (e ValidationError) Error() string { return string(e) }
