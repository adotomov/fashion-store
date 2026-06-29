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
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router, optionalAuth func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(optionalAuth)
		r.Get("/checkout/delivery-methods", h.listDeliveryMethods)
		r.Post("/checkout", h.placeOrder)
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
	httpx.WriteJSON(w, http.StatusCreated, toOrderResultResponse(result))
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrCartEmpty):
		httpx.WriteError(w, http.StatusBadRequest, "cart_empty", "your cart is empty")
	case errors.Is(err, domain.ErrInvalidDeliveryMethod):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_delivery_method", "delivery method is invalid")
	case errors.Is(err, domain.ErrInvalidPaymentMethod):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_payment_method", "payment method is invalid")
	case errors.Is(err, domain.ErrDeliveryMethodUnavailable):
		httpx.WriteError(w, http.StatusBadRequest, "delivery_method_unavailable", "this delivery method is currently unavailable")
	case errors.Is(err, domain.ErrOfficeRequired):
		httpx.WriteError(w, http.StatusBadRequest, "office_required", "a pickup locker is required for this delivery method")
	case errors.Is(err, domain.ErrInsufficientStock):
		httpx.WriteError(w, http.StatusConflict, "insufficient_stock", "not enough stock available")
	case errors.Is(err, domain.ErrPaymentFailed):
		httpx.WriteError(w, http.StatusPaymentRequired, "payment_failed", "payment could not be processed")
	case errors.As(err, new(domain.ValidationError)):
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", err.Error())
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
