package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/inventory/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/inventory/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/authctx"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router, requireAdmin func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(requireAdmin)
		r.Route("/admin/inventory/items", func(r chi.Router) {
			r.Get("/", h.list)
			r.Post("/", h.create)
			r.Get("/{id}", h.get)
			r.Patch("/{id}", h.updateSKU)
			r.Post("/{id}/adjust", h.adjust)
			r.Get("/{id}/movements", h.listMovements)
		})
	})
}

const timeFormat = "2006-01-02T15:04:05Z07:00"

type itemResponse struct {
	ID                string `json:"id"`
	VariantID         string `json:"variant_id"`
	SKU               string `json:"sku"`
	QuantityOnHand    int    `json:"quantity_on_hand"`
	QuantityReserved  int    `json:"quantity_reserved"`
	QuantityAvailable int    `json:"quantity_available"`
	ProductName       string `json:"product_name,omitempty"`
	VariantLabel      string `json:"variant_label,omitempty"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

func toItemResponse(i domain.InventoryItem) itemResponse {
	return itemResponse{
		ID:                i.ID.String(),
		VariantID:         i.VariantID.String(),
		SKU:               i.SKU,
		QuantityOnHand:    i.QuantityOnHand,
		QuantityReserved:  i.QuantityReserved,
		QuantityAvailable: i.QuantityAvailable(),
		ProductName:       i.ProductName,
		VariantLabel:      i.VariantLabel,
		CreatedAt:         i.CreatedAt.Format(timeFormat),
		UpdatedAt:         i.UpdatedAt.Format(timeFormat),
	}
}

type movementResponse struct {
	ID            string  `json:"id"`
	Type          string  `json:"type"`
	QuantityDelta int     `json:"quantity_delta"`
	Note          string  `json:"note"`
	CreatedBy     *string `json:"created_by,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

func toMovementResponse(m domain.InventoryMovement) movementResponse {
	resp := movementResponse{
		ID:            m.ID.String(),
		Type:          string(m.Type),
		QuantityDelta: m.QuantityDelta,
		Note:          m.Note,
		CreatedAt:     m.CreatedAt.Format(timeFormat),
	}
	if m.CreatedBy != nil {
		s := m.CreatedBy.String()
		resp.CreatedBy = &s
	}
	return resp
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		VariantID       string `json:"variant_id"`
		SKU             string `json:"sku"`
		InitialQuantity int    `json:"initial_quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	variantID, err := uuid.Parse(req.VariantID)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_variant_id", "variant_id is invalid")
		return
	}

	createdBy := adminUserID(r)
	item, err := h.service.CreateItem(r.Context(), variantID, req.SKU, req.InitialQuantity, createdBy)
	if err != nil {
		writeInventoryError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toItemResponse(*item))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListItems(r.Context())
	if err != nil {
		writeInventoryError(w, err)
		return
	}

	resp := make([]itemResponse, 0, len(items))
	for _, i := range items {
		resp = append(resp, toItemResponse(i))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "item id is invalid")
		return
	}

	item, err := h.service.GetItem(r.Context(), id)
	if err != nil {
		writeInventoryError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toItemResponse(*item))
}

func (h *Handler) updateSKU(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "item id is invalid")
		return
	}

	var req struct {
		SKU string `json:"sku"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	item, err := h.service.UpdateSKU(r.Context(), id, req.SKU)
	if err != nil {
		writeInventoryError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toItemResponse(*item))
}

func (h *Handler) adjust(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "item id is invalid")
		return
	}

	var req struct {
		Type          string `json:"type"`
		QuantityDelta int    `json:"quantity_delta"`
		Note          string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	createdBy := adminUserID(r)
	item, err := h.service.AdjustStock(r.Context(), id, domain.MovementType(req.Type), req.QuantityDelta, req.Note, createdBy)
	if err != nil {
		writeInventoryError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toItemResponse(*item))
}

func (h *Handler) listMovements(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "item id is invalid")
		return
	}

	movements, err := h.service.ListMovements(r.Context(), id)
	if err != nil {
		writeInventoryError(w, err)
		return
	}

	resp := make([]movementResponse, 0, len(movements))
	for _, m := range movements {
		resp = append(resp, toMovementResponse(m))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func adminUserID(r *http.Request) *uuid.UUID {
	principal, ok := authctx.FromContext(r.Context())
	if !ok {
		return nil
	}
	id := principal.UserID
	return &id
}

func writeInventoryError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrItemNotFound):
		httpx.WriteError(w, http.StatusNotFound, "item_not_found", "inventory item not found")
	case errors.Is(err, domain.ErrVariantNotFound):
		httpx.WriteError(w, http.StatusBadRequest, "variant_not_found", "product variant not found")
	case errors.Is(err, domain.ErrItemAlreadyExists):
		httpx.WriteError(w, http.StatusConflict, "item_already_exists", "inventory item already exists for this variant")
	case errors.Is(err, domain.ErrSKUConflict):
		httpx.WriteError(w, http.StatusConflict, "sku_conflict", "sku is already in use")
	case errors.Is(err, domain.ErrInvalidMovementType):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_movement_type", "movement type is invalid or not admin-adjustable")
	case errors.Is(err, domain.ErrInsufficientStock):
		httpx.WriteError(w, http.StatusBadRequest, "insufficient_stock", "adjustment would result in negative stock on hand")
	case errors.As(err, new(domain.ValidationError)):
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", err.Error())
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
