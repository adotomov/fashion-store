package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/cart/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/cart/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/authctx"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
)

// cartTokenHeader carries the opaque guest-cart identifier for anonymous
// callers. Authenticated callers are identified via their bearer principal
// instead and don't need it.
const cartTokenHeader = "X-Cart-Token"

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router, optionalAuth, requireAuth func(http.Handler) http.Handler) {
	r.Route("/cart", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(optionalAuth)
			r.Get("/", h.getCart)
			r.Post("/items", h.addItem)
			r.Patch("/items/{itemId}", h.updateItem)
			r.Delete("/items/{itemId}", h.removeItem)
		})
		r.Group(func(r chi.Router) {
			r.Use(requireAuth)
			r.Post("/merge", h.merge)
		})
	})
}

func ownerFromRequest(r *http.Request) (application.CartOwner, error) {
	if p, ok := authctx.FromContext(r.Context()); ok {
		return application.CartOwner{UserID: &p.UserID}, nil
	}
	if raw := r.Header.Get(cartTokenHeader); raw != "" {
		token, err := uuid.Parse(raw)
		if err != nil {
			return application.CartOwner{}, err
		}
		return application.CartOwner{GuestToken: &token}, nil
	}
	return application.CartOwner{}, nil
}

func (h *Handler) getCart(w http.ResponseWriter, r *http.Request) {
	owner, err := ownerFromRequest(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_cart_token", "cart token is invalid")
		return
	}

	cart, err := h.service.GetCart(r.Context(), owner)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toCartResponse(*cart))
}

type addItemRequest struct {
	VariantID string `json:"variant_id"`
	Quantity  int    `json:"quantity"`
}

func (h *Handler) addItem(w http.ResponseWriter, r *http.Request) {
	owner, err := ownerFromRequest(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_cart_token", "cart token is invalid")
		return
	}

	var req addItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}
	variantID, err := uuid.Parse(req.VariantID)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_variant_id", "variant_id is invalid")
		return
	}

	cart, err := h.service.AddItem(r.Context(), owner, variantID, req.Quantity)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toCartResponse(*cart))
}

type updateItemRequest struct {
	Quantity int `json:"quantity"`
}

func (h *Handler) updateItem(w http.ResponseWriter, r *http.Request) {
	owner, err := ownerFromRequest(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_cart_token", "cart token is invalid")
		return
	}

	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "item id is invalid")
		return
	}

	var req updateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	cart, err := h.service.UpdateItemQuantity(r.Context(), owner, itemID, req.Quantity)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toCartResponse(*cart))
}

func (h *Handler) removeItem(w http.ResponseWriter, r *http.Request) {
	owner, err := ownerFromRequest(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_cart_token", "cart token is invalid")
		return
	}

	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "item id is invalid")
		return
	}

	cart, err := h.service.RemoveItem(r.Context(), owner, itemID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toCartResponse(*cart))
}

type mergeRequest struct {
	GuestToken string `json:"guest_token"`
}

func (h *Handler) merge(w http.ResponseWriter, r *http.Request) {
	p, ok := authctx.FromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing principal")
		return
	}

	var req mergeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}
	guestToken, err := uuid.Parse(req.GuestToken)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_guest_token", "guest_token is invalid")
		return
	}

	cart, err := h.service.MergeGuestCartIntoUser(r.Context(), p.UserID, guestToken)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toCartResponse(*cart))
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrCartNotFound):
		httpx.WriteError(w, http.StatusNotFound, "cart_not_found", "cart not found")
	case errors.Is(err, domain.ErrCartItemNotFound):
		httpx.WriteError(w, http.StatusNotFound, "cart_item_not_found", "cart item not found")
	case errors.Is(err, domain.ErrVariantNotFound):
		httpx.WriteError(w, http.StatusNotFound, "variant_not_found", "product variant not found")
	case errors.Is(err, domain.ErrInsufficientStock):
		httpx.WriteError(w, http.StatusConflict, "insufficient_stock", "not enough stock available")
	case errors.As(err, new(domain.ValidationError)):
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", err.Error())
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
