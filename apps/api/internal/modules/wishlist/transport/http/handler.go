package http

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/wishlist/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/wishlist/domain"
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
	r.Route("/wishlist", func(r chi.Router) {
		r.Use(requireAuth)
		r.Get("/", h.list)
		r.Post("/{productId}", h.add)
		r.Delete("/{productId}", h.remove)
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

	items, err := h.service.List(r.Context(), p.UserID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	resp := make([]itemResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, toItemResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) add(w http.ResponseWriter, r *http.Request) {
	p, ok := principalFrom(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing principal")
		return
	}

	productID, err := uuid.Parse(chi.URLParam(r, "productId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "product id is invalid")
		return
	}

	item, err := h.service.Add(r.Context(), p.UserID, productID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toItemResponse(*item))
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	p, ok := principalFrom(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing principal")
		return
	}

	productID, err := uuid.Parse(chi.URLParam(r, "productId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "product id is invalid")
		return
	}

	if err := h.service.Remove(r.Context(), p.UserID, productID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrItemNotFound):
		httpx.WriteError(w, http.StatusNotFound, "wishlist_item_not_found", "wishlist item not found")
	case errors.Is(err, domain.ErrProductNotFound):
		httpx.WriteError(w, http.StatusNotFound, "product_not_found", "product not found")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
