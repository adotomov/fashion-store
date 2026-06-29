package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/payments/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/payments/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/authctx"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router, requireAuth func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Get("/me/payment-methods", h.list)
		r.Post("/me/payment-methods", h.create)
		r.Patch("/me/payment-methods/{id}", h.update)
		r.Delete("/me/payment-methods/{id}", h.delete)
	})
}

func principalFrom(r *http.Request) (authctx.Principal, bool) {
	return authctx.FromContext(r.Context())
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	p, ok := principalFrom(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing principal")
		return
	}

	methods, err := h.service.ListPaymentMethods(r.Context(), p.UserID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	resp := make([]paymentMethodResponse, 0, len(methods))
	for _, m := range methods {
		resp = append(resp, toPaymentMethodResponse(m))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

type paymentMethodRequest struct {
	Brand     string `json:"brand"`
	Last4     string `json:"last4"`
	ExpMonth  int    `json:"exp_month"`
	ExpYear   int    `json:"exp_year"`
	IsDefault bool   `json:"is_default"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	p, ok := principalFrom(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing principal")
		return
	}

	var req paymentMethodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	method, err := h.service.AddPaymentMethod(r.Context(), p.UserID, application.CreatePaymentMethodInput{
		Brand:     req.Brand,
		Last4:     req.Last4,
		ExpMonth:  req.ExpMonth,
		ExpYear:   req.ExpYear,
		IsDefault: req.IsDefault,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toPaymentMethodResponse(*method))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	p, ok := principalFrom(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing principal")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "payment method id is invalid")
		return
	}

	var req struct {
		Brand     *string `json:"brand,omitempty"`
		Last4     *string `json:"last4,omitempty"`
		ExpMonth  *int    `json:"exp_month,omitempty"`
		ExpYear   *int    `json:"exp_year,omitempty"`
		IsDefault *bool   `json:"is_default,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	method, err := h.service.UpdatePaymentMethod(r.Context(), p.UserID, id, application.UpdatePaymentMethodInput{
		Brand:     req.Brand,
		Last4:     req.Last4,
		ExpMonth:  req.ExpMonth,
		ExpYear:   req.ExpYear,
		IsDefault: req.IsDefault,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toPaymentMethodResponse(*method))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	p, ok := principalFrom(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing principal")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "payment method id is invalid")
		return
	}

	if err := h.service.DeletePaymentMethod(r.Context(), p.UserID, id); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrPaymentMethodNotFound):
		httpx.WriteError(w, http.StatusNotFound, "payment_method_not_found", "payment method not found")
	case errors.As(err, new(domain.ValidationError)):
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", err.Error())
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
