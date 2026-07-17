package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/orders/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/orders/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/authctx"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router, requireAuth, requireAdmin func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Get("/me/orders", h.listOrders)
	})

	r.Group(func(r chi.Router) {
		r.Use(requireAdmin)
		r.Get("/admin/orders", h.adminListOrders)
		r.Get("/admin/orders/stats", h.adminOrderStats)
		r.Get("/admin/orders/unviewed-count", h.adminUnviewedCount)
		r.Get("/admin/orders/{id}", h.adminGetOrder)
		r.Get("/admin/orders/{id}/transactions", h.adminOrderTransactions)
		r.Patch("/admin/orders/{id}", h.adminUpdateFulfillment)
	})
}

func (h *Handler) adminOrderTransactions(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "order id is invalid")
		return
	}
	txns, err := h.service.ListPaymentTransactions(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toPaymentTransactionResponses(txns))
}

func (h *Handler) listOrders(w http.ResponseWriter, r *http.Request) {
	p, ok := authctx.FromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing principal")
		return
	}

	orders, err := h.service.ListOrders(r.Context(), p.UserID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	resp := make([]orderResponse, 0, len(orders))
	for _, o := range orders {
		resp = append(resp, toOrderResponse(o))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) adminListOrders(w http.ResponseWriter, r *http.Request) {
	filter := application.AdminListOrdersFilter{}
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = &status
	}
	if r.URL.Query().Get("unviewed_only") == "true" {
		filter.UnviewedOnly = true
	}

	orders, err := h.service.AdminListOrders(r.Context(), filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	resp := make([]orderResponse, 0, len(orders))
	for _, o := range orders {
		resp = append(resp, toOrderResponse(o))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) adminGetOrder(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "order id is invalid")
		return
	}

	order, err := h.service.AdminGetOrder(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toOrderResponse(*order))
}

type updateFulfillmentRequest struct {
	Status         *string `json:"status,omitempty"`
	Carrier        *string `json:"carrier,omitempty"`
	TrackingNumber *string `json:"tracking_number,omitempty"`
	ShipmentStatus *string `json:"shipment_status,omitempty"`
}

func (h *Handler) adminUpdateFulfillment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "order id is invalid")
		return
	}

	var req updateFulfillmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	order, err := h.service.UpdateFulfillment(r.Context(), id, application.UpdateFulfillmentInput{
		Status:         req.Status,
		Carrier:        req.Carrier,
		TrackingNumber: req.TrackingNumber,
		ShipmentStatus: req.ShipmentStatus,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toOrderResponse(*order))
}

func (h *Handler) adminOrderStats(w http.ResponseWriter, r *http.Request) {
	var since time.Time
	switch r.URL.Query().Get("range") {
	case "30d":
		since = time.Now().AddDate(0, 0, -30)
	case "90d":
		since = time.Now().AddDate(0, 0, -90)
	default:
		since = time.Now().AddDate(0, 0, -7)
	}

	stats, err := h.service.OrderStats(r.Context(), since)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toOrderStatsResponse(stats))
}

func (h *Handler) adminUnviewedCount(w http.ResponseWriter, r *http.Request) {
	count, err := h.service.CountUnviewedOrders(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, struct {
		Count int `json:"count"`
	}{Count: count})
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrOrderNotFound):
		httpx.WriteError(w, http.StatusNotFound, "order_not_found", "order not found")
	case errors.As(err, new(domain.ValidationError)):
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", err.Error())
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
