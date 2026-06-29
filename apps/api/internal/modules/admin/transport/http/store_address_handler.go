package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/admin/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/admin/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
)

type StoreAddressHandler struct {
	service *application.StoreAddressService
}

func NewStoreAddressHandler(service *application.StoreAddressService) *StoreAddressHandler {
	return &StoreAddressHandler{service: service}
}

func (h *StoreAddressHandler) RegisterRoutes(r chi.Router, requireAdmin func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(requireAdmin)
		r.Route("/admin/store-settings/addresses", func(r chi.Router) {
			r.Get("/", h.list)
			r.Post("/", h.create)
			r.Patch("/{id}", h.update)
			r.Delete("/{id}", h.delete)
		})
	})
}

func (h *StoreAddressHandler) RegisterStorefrontRoutes(r chi.Router) {
	r.Get("/storefront/store-settings/addresses", h.listPublic)
}

type storeAddressResponse struct {
	ID         string  `json:"id"`
	Label      string  `json:"label"`
	Line1      string  `json:"line1"`
	Line2      *string `json:"line2,omitempty"`
	City       *string `json:"city,omitempty"`
	Region     *string `json:"region,omitempty"`
	PostalCode *string `json:"postal_code,omitempty"`
	Country    *string `json:"country,omitempty"`
	IsDefault  bool    `json:"is_default"`
}

func toStoreAddressResponse(a domain.StoreAddress) storeAddressResponse {
	return storeAddressResponse{
		ID:         a.ID.String(),
		Label:      a.Label,
		Line1:      a.Line1,
		Line2:      a.Line2,
		City:       a.City,
		Region:     a.Region,
		PostalCode: a.PostalCode,
		Country:    a.Country,
		IsDefault:  a.IsDefault,
	}
}

func (h *StoreAddressHandler) list(w http.ResponseWriter, r *http.Request) {
	addresses, err := h.service.List(r.Context())
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	resp := make([]storeAddressResponse, 0, len(addresses))
	for _, a := range addresses {
		resp = append(resp, toStoreAddressResponse(a))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *StoreAddressHandler) listPublic(w http.ResponseWriter, r *http.Request) {
	h.list(w, r)
}

type upsertStoreAddressRequest struct {
	Label      string  `json:"label"`
	Line1      string  `json:"line1"`
	Line2      *string `json:"line2,omitempty"`
	City       *string `json:"city,omitempty"`
	Region     *string `json:"region,omitempty"`
	PostalCode *string `json:"postal_code,omitempty"`
	Country    *string `json:"country,omitempty"`
	IsDefault  bool    `json:"is_default"`
}

func (req upsertStoreAddressRequest) toInput() application.UpsertStoreAddressInput {
	return application.UpsertStoreAddressInput{
		Label:      req.Label,
		Line1:      req.Line1,
		Line2:      req.Line2,
		City:       req.City,
		Region:     req.Region,
		PostalCode: req.PostalCode,
		Country:    req.Country,
		IsDefault:  req.IsDefault,
	}
}

func (h *StoreAddressHandler) create(w http.ResponseWriter, r *http.Request) {
	var req upsertStoreAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}
	address, err := h.service.Create(r.Context(), req.toInput())
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toStoreAddressResponse(*address))
}

func (h *StoreAddressHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "address id must be a UUID")
		return
	}
	var req upsertStoreAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}
	address, err := h.service.Update(r.Context(), id, req.toInput())
	if err != nil {
		writeAdminModuleError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toStoreAddressResponse(*address))
}

func (h *StoreAddressHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "address id must be a UUID")
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		writeAdminModuleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
