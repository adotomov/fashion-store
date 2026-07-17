package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/checkout/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/checkout/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/authctx"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
)

// cartTokenHeader matches the cart module's own header name — checkout
// resolves the same guest cart the customer has been adding items to.
const cartTokenHeader = "X-Cart-Token"

type Handler struct {
	service       *application.Service
	webhookSecret string
}

func NewHandler(service *application.Service, webhookSecret string) *Handler {
	return &Handler{service: service, webhookSecret: webhookSecret}
}

func (h *Handler) RegisterRoutes(r chi.Router, optionalAuth, requireAdmin func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(optionalAuth)
		r.Get("/checkout/delivery-methods", h.listDeliveryMethods)
		r.Post("/checkout", h.placeOrder)
		r.Get("/checkout/orders/{order_number}/status", h.orderStatus)
	})
	r.Group(func(r chi.Router) {
		r.Use(requireAdmin)
		r.Post("/admin/orders/{id}/refund", h.refundOrder)
	})
}

func (h *Handler) listDeliveryMethods(w http.ResponseWriter, r *http.Request) {
	methods := h.service.ListDeliveryMethods(r.Context())
	resp := make([]deliveryMethodResponse, 0, len(methods))
	for _, m := range methods {
		resp = append(resp, toDeliveryMethodResponse(m))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func ownerFromRequest(r *http.Request) (application.CartOwner, *uuid.UUID, error) {
	if p, ok := authctx.FromContext(r.Context()); ok {
		return application.CartOwner{UserID: &p.UserID}, &p.UserID, nil
	}
	if raw := r.Header.Get(cartTokenHeader); raw != "" {
		token, err := uuid.Parse(raw)
		if err != nil {
			return application.CartOwner{}, nil, err
		}
		return application.CartOwner{GuestToken: &token}, nil, nil
	}
	return application.CartOwner{}, nil, nil
}

func (h *Handler) placeOrder(w http.ResponseWriter, r *http.Request) {
	owner, principalUserID, err := ownerFromRequest(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_cart_token", "cart token is invalid")
		return
	}

	var req placeOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	result, err := h.service.PlaceOrder(r.Context(), owner, principalUserID, req.toInput())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	// Online-card orders come back awaiting widget payment; pay-on-delivery
	// orders come back fully placed.
	if result.PaymentRequired != nil {
		httpx.WriteJSON(w, http.StatusCreated, toPaymentInitiationResponse(*result.PaymentRequired))
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toOrderResultResponse(*result.Order))
}

// orderStatus is the public post-payment poll: it returns only the order
// status (keyed by order number), enough for the confirmation page to reflect
// the webhook-driven settlement without exposing order details.
func (h *Handler) orderStatus(w http.ResponseWriter, r *http.Request) {
	orderNumber := chi.URLParam(r, "order_number")
	status, err := h.service.PaymentStatus(r.Context(), orderNumber)
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "order_not_found", "order not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"order_number": orderNumber, "status": status})
}

func (h *Handler) refundOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_order_id", "order id is invalid")
		return
	}

	var req refundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	var adminID *uuid.UUID
	if p, ok := authctx.FromContext(r.Context()); ok {
		adminID = &p.UserID
	}

	if err := h.service.RefundOrder(r.Context(), orderID, req.AmountMinor, req.Reason, adminID); err != nil {
		writeRefundError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeRefundError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrOrderNotFound):
		httpx.WriteError(w, http.StatusNotFound, "order_not_found", "order not found")
	case errors.Is(err, domain.ErrRefundNotAllowed):
		httpx.WriteError(w, http.StatusConflict, "refund_not_allowed", "this order cannot be refunded")
	case errors.Is(err, domain.ErrRefundAmountInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "refund_amount_invalid", "refund amount is invalid")
	case errors.Is(err, domain.ErrRefundFailed):
		httpx.WriteError(w, http.StatusBadGateway, "refund_failed", "the refund could not be processed")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrCartEmpty):
		httpx.WriteError(w, http.StatusBadRequest, "cart_empty", "your cart is empty")
	case errors.Is(err, domain.ErrInvalidDeliveryMethod):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_delivery_method", "delivery method is invalid")
	case errors.Is(err, domain.ErrInvalidPaymentMethod):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_payment_method", "payment method is invalid")
	case errors.Is(err, domain.ErrPaymentMethodNotAllowed):
		httpx.WriteError(w, http.StatusBadRequest, "payment_method_not_allowed", "this payment method is not available for the chosen delivery method")
	case errors.Is(err, domain.ErrDeliveryMethodUnavailable):
		httpx.WriteError(w, http.StatusBadRequest, "delivery_method_unavailable", "this delivery method is currently unavailable")
	case errors.Is(err, domain.ErrOfficeRequired):
		httpx.WriteError(w, http.StatusBadRequest, "office_required", "a pickup locker is required for this delivery method")
	case errors.Is(err, domain.ErrInvalidDiscountCode):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_discount_code", "discount code is invalid, expired, or exhausted")
	case errors.Is(err, domain.ErrInsufficientStock):
		httpx.WriteError(w, http.StatusConflict, "insufficient_stock", "not enough stock available")
	case errors.Is(err, domain.ErrPaymentFailed):
		httpx.WriteError(w, http.StatusPaymentRequired, "payment_failed", "payment could not be processed")
	case errors.Is(err, domain.ErrPaymentInitiation):
		httpx.WriteError(w, http.StatusBadGateway, "payment_initiation_failed", "could not start card payment, please try again")
	case errors.As(err, new(domain.ValidationError)):
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", err.Error())
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
