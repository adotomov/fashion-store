package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/promotions/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/promotions/domain"
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

		r.Get("/admin/promotions", h.listPromotions)
		r.Post("/admin/promotions", h.createPromotion)
		r.Get("/admin/promotions/{id}", h.getPromotion)
		r.Patch("/admin/promotions/{id}", h.updatePromotion)
		r.Delete("/admin/promotions/{id}", h.deletePromotion)

		r.Get("/admin/discount-codes", h.listCodes)
		r.Post("/admin/discount-codes", h.createCode)
		r.Get("/admin/discount-codes/{id}", h.getCode)
		r.Patch("/admin/discount-codes/{id}", h.updateCode)
		r.Delete("/admin/discount-codes/{id}", h.deleteCode)
	})

	// Public endpoint for checkout to validate a discount code preview.
	r.Get("/checkout/discount", h.validateCode)
}

// --- Promotions ---

type promotionResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	TargetType  string   `json:"target_type"`
	CategoryIDs []string `json:"category_ids"`
	TypeIDs     []string `json:"type_ids"`
	ProductIDs  []string `json:"product_ids"`

	ValuePercent       *int    `json:"value_percent,omitempty"`
	ValueFixedMinor    *int64  `json:"value_fixed_minor,omitempty"`
	ValueFixedCurrency *string `json:"value_fixed_currency,omitempty"`
	BuyQty             *int    `json:"buy_qty,omitempty"`
	GetQty             *int    `json:"get_qty,omitempty"`
	GetDiscountPct     *int    `json:"get_discount_pct,omitempty"`
	MinQuantity        int     `json:"min_quantity"`

	StartsAt  *string `json:"starts_at,omitempty"`
	EndsAt    *string `json:"ends_at,omitempty"`
	IsActive  bool    `json:"is_active"`
	Priority  int     `json:"priority"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func toPromotionResponse(p domain.Promotion) promotionResponse {
	r := promotionResponse{
		ID:                 p.ID.String(),
		Name:               p.Name,
		Description:        p.Description,
		Type:               string(p.Type),
		TargetType:         string(p.TargetType),
		CategoryIDs:        uuidsToStrings(p.CategoryIDs),
		TypeIDs:            uuidsToStrings(p.TypeIDs),
		ProductIDs:         uuidsToStrings(p.ProductIDs),
		ValuePercent:       p.ValuePercent,
		ValueFixedMinor:    p.ValueFixedMinor,
		ValueFixedCurrency: p.ValueFixedCurrency,
		BuyQty:             p.BuyQty,
		GetQty:             p.GetQty,
		GetDiscountPct:     p.GetDiscountPct,
		MinQuantity:        p.MinQuantity,
		IsActive:           p.IsActive,
		Priority:           p.Priority,
		CreatedAt:          p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          p.UpdatedAt.Format(time.RFC3339),
	}
	if p.StartsAt != nil {
		s := p.StartsAt.Format(time.RFC3339)
		r.StartsAt = &s
	}
	if p.EndsAt != nil {
		s := p.EndsAt.Format(time.RFC3339)
		r.EndsAt = &s
	}
	return r
}

func uuidsToStrings(ids []uuid.UUID) []string {
	if ids == nil {
		return []string{}
	}
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

func (h *Handler) listPromotions(w http.ResponseWriter, r *http.Request) {
	promotions, err := h.service.ListPromotions(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	resp := make([]promotionResponse, 0, len(promotions))
	for _, p := range promotions {
		resp = append(resp, toPromotionResponse(p))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) getPromotion(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	p, err := h.service.GetPromotion(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toPromotionResponse(*p))
}

type createPromotionRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Type        string  `json:"type"`
	TargetType  string  `json:"target_type"`
	CategoryIDs []string `json:"category_ids"`
	TypeIDs     []string `json:"type_ids"`
	ProductIDs  []string `json:"product_ids"`

	ValuePercent       *int    `json:"value_percent"`
	ValueFixedMinor    *int64  `json:"value_fixed_minor"`
	ValueFixedCurrency *string `json:"value_fixed_currency"`
	BuyQty             *int    `json:"buy_qty"`
	GetQty             *int    `json:"get_qty"`
	GetDiscountPct     *int    `json:"get_discount_pct"`
	MinQuantity        int     `json:"min_quantity"`

	StartsAt *string `json:"starts_at"`
	EndsAt   *string `json:"ends_at"`
	IsActive bool    `json:"is_active"`
	Priority int     `json:"priority"`
}

func (req createPromotionRequest) toInput() (application.CreatePromotionInput, error) {
	inp := application.CreatePromotionInput{
		Name:               req.Name,
		Description:        req.Description,
		Type:               domain.PromotionType(req.Type),
		TargetType:         domain.TargetType(req.TargetType),
		ValuePercent:       req.ValuePercent,
		ValueFixedMinor:    req.ValueFixedMinor,
		ValueFixedCurrency: req.ValueFixedCurrency,
		BuyQty:             req.BuyQty,
		GetQty:             req.GetQty,
		GetDiscountPct:     req.GetDiscountPct,
		MinQuantity:        req.MinQuantity,
		IsActive:           req.IsActive,
		Priority:           req.Priority,
	}
	var err error
	inp.CategoryIDs, err = parseUUIDs(req.CategoryIDs)
	if err != nil {
		return inp, err
	}
	inp.TypeIDs, err = parseUUIDs(req.TypeIDs)
	if err != nil {
		return inp, err
	}
	inp.ProductIDs, err = parseUUIDs(req.ProductIDs)
	if err != nil {
		return inp, err
	}
	if req.StartsAt != nil {
		t, err := time.Parse(time.RFC3339, *req.StartsAt)
		if err != nil {
			return inp, err
		}
		inp.StartsAt = &t
	}
	if req.EndsAt != nil {
		t, err := time.Parse(time.RFC3339, *req.EndsAt)
		if err != nil {
			return inp, err
		}
		inp.EndsAt = &t
	}
	return inp, nil
}

func (h *Handler) createPromotion(w http.ResponseWriter, r *http.Request) {
	var req createPromotionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}
	inp, err := req.toInput()
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_fields", err.Error())
		return
	}
	p, err := h.service.CreatePromotion(r.Context(), inp)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toPromotionResponse(*p))
}

type updatePromotionRequest struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Type        *string  `json:"type"`
	TargetType  *string  `json:"target_type"`
	CategoryIDs *[]string `json:"category_ids"`
	TypeIDs     *[]string `json:"type_ids"`
	ProductIDs  *[]string `json:"product_ids"`

	ValuePercent       *int    `json:"value_percent"`
	ClearValuePercent  bool    `json:"clear_value_percent"`
	ValueFixedMinor    *int64  `json:"value_fixed_minor"`
	ValueFixedCurrency *string `json:"value_fixed_currency"`
	ClearFixed         bool    `json:"clear_fixed"`
	BuyQty             *int    `json:"buy_qty"`
	GetQty             *int    `json:"get_qty"`
	GetDiscountPct     *int    `json:"get_discount_pct"`
	ClearBxgy          bool    `json:"clear_bxgy"`
	MinQuantity        *int    `json:"min_quantity"`

	StartsAt    *string `json:"starts_at"`
	ClearStarts bool    `json:"clear_starts"`
	EndsAt      *string `json:"ends_at"`
	ClearEnds   bool    `json:"clear_ends"`
	IsActive    *bool   `json:"is_active"`
	Priority    *int    `json:"priority"`
}

func (h *Handler) updatePromotion(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	var req updatePromotionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	inp := application.UpdatePromotionInput{
		Name:               req.Name,
		ValuePercent:       req.ValuePercent,
		ClearValuePercent:  req.ClearValuePercent,
		ValueFixedMinor:    req.ValueFixedMinor,
		ValueFixedCurrency: req.ValueFixedCurrency,
		ClearFixed:         req.ClearFixed,
		BuyQty:             req.BuyQty,
		GetQty:             req.GetQty,
		GetDiscountPct:     req.GetDiscountPct,
		ClearBxgy:          req.ClearBxgy,
		MinQuantity:        req.MinQuantity,
		ClearStarts:        req.ClearStarts,
		ClearEnds:          req.ClearEnds,
		IsActive:           req.IsActive,
		Priority:           req.Priority,
	}
	if req.Description != nil {
		inp.Description = req.Description
	}
	if req.Type != nil {
		t := domain.PromotionType(*req.Type)
		inp.Type = &t
	}
	if req.TargetType != nil {
		t := domain.TargetType(*req.TargetType)
		inp.TargetType = &t
	}
	if req.CategoryIDs != nil {
		ids, err := parseUUIDs(*req.CategoryIDs)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_fields", "invalid category_ids")
			return
		}
		inp.CategoryIDs = &ids
	}
	if req.TypeIDs != nil {
		ids, err := parseUUIDs(*req.TypeIDs)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_fields", "invalid type_ids")
			return
		}
		inp.TypeIDs = &ids
	}
	if req.ProductIDs != nil {
		ids, err := parseUUIDs(*req.ProductIDs)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_fields", "invalid product_ids")
			return
		}
		inp.ProductIDs = &ids
	}
	if req.StartsAt != nil {
		t, err := time.Parse(time.RFC3339, *req.StartsAt)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_fields", "invalid starts_at")
			return
		}
		inp.StartsAt = &t
	}
	if req.EndsAt != nil {
		t, err := time.Parse(time.RFC3339, *req.EndsAt)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_fields", "invalid ends_at")
			return
		}
		inp.EndsAt = &t
	}

	p, err := h.service.UpdatePromotion(r.Context(), id, inp)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toPromotionResponse(*p))
}

func (h *Handler) deletePromotion(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	if err := h.service.DeletePromotion(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Discount Codes ---

type codeResponse struct {
	ID           string  `json:"id"`
	Code         string  `json:"code"`
	ValuePercent int     `json:"value_percent"`
	StartsAt     *string `json:"starts_at,omitempty"`
	ExpiresAt    *string `json:"expires_at,omitempty"`
	MaxUses      *int    `json:"max_uses,omitempty"`
	UseCount     int     `json:"use_count"`
	IsActive     bool    `json:"is_active"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

func toCodeResponse(c domain.DiscountCode) codeResponse {
	r := codeResponse{
		ID:           c.ID.String(),
		Code:         c.Code,
		ValuePercent: c.ValuePercent,
		MaxUses:      c.MaxUses,
		UseCount:     c.UseCount,
		IsActive:     c.IsActive,
		CreatedAt:    c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    c.UpdatedAt.Format(time.RFC3339),
	}
	if c.StartsAt != nil {
		s := c.StartsAt.Format(time.RFC3339)
		r.StartsAt = &s
	}
	if c.ExpiresAt != nil {
		s := c.ExpiresAt.Format(time.RFC3339)
		r.ExpiresAt = &s
	}
	return r
}

func (h *Handler) listCodes(w http.ResponseWriter, r *http.Request) {
	codes, err := h.service.ListCodes(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	resp := make([]codeResponse, 0, len(codes))
	for _, c := range codes {
		resp = append(resp, toCodeResponse(c))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) getCode(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	c, err := h.service.GetCode(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toCodeResponse(*c))
}

type createCodeRequest struct {
	Code         string  `json:"code"`
	ValuePercent int     `json:"value_percent"`
	StartsAt     *string `json:"starts_at"`
	ExpiresAt    *string `json:"expires_at"`
	MaxUses      *int    `json:"max_uses"`
	IsActive     bool    `json:"is_active"`
}

func (h *Handler) createCode(w http.ResponseWriter, r *http.Request) {
	var req createCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}
	inp := application.CreateCodeInput{
		Code:         req.Code,
		ValuePercent: req.ValuePercent,
		MaxUses:      req.MaxUses,
		IsActive:     req.IsActive,
	}
	if req.StartsAt != nil {
		t, err := time.Parse(time.RFC3339, *req.StartsAt)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_fields", "invalid starts_at")
			return
		}
		inp.StartsAt = &t
	}
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_fields", "invalid expires_at")
			return
		}
		inp.ExpiresAt = &t
	}
	c, err := h.service.CreateCode(r.Context(), inp)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toCodeResponse(*c))
}

type updateCodeRequest struct {
	ValuePercent *int    `json:"value_percent"`
	StartsAt     *string `json:"starts_at"`
	ClearStarts  bool    `json:"clear_starts"`
	ExpiresAt    *string `json:"expires_at"`
	ClearExpiry  bool    `json:"clear_expiry"`
	MaxUses      *int    `json:"max_uses"`
	ClearMaxUses bool    `json:"clear_max_uses"`
	IsActive     *bool   `json:"is_active"`
}

func (h *Handler) updateCode(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	var req updateCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}
	inp := application.UpdateCodeInput{
		ValuePercent: req.ValuePercent,
		ClearStarts:  req.ClearStarts,
		ClearExpiry:  req.ClearExpiry,
		MaxUses:      req.MaxUses,
		ClearMaxUses: req.ClearMaxUses,
		IsActive:     req.IsActive,
	}
	if req.StartsAt != nil {
		t, err := time.Parse(time.RFC3339, *req.StartsAt)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_fields", "invalid starts_at")
			return
		}
		inp.StartsAt = &t
	}
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_fields", "invalid expires_at")
			return
		}
		inp.ExpiresAt = &t
	}
	c, err := h.service.UpdateCode(r.Context(), id, inp)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toCodeResponse(*c))
}

func (h *Handler) deleteCode(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid id")
		return
	}
	if err := h.service.DeleteCode(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type validateCodeResponse struct {
	Code         string `json:"code"`
	ValuePercent int    `json:"value_percent"`
	Valid         bool   `json:"valid"`
}

func (h *Handler) validateCode(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		httpx.WriteError(w, http.StatusBadRequest, "missing_code", "code query parameter is required")
		return
	}
	dc, err := h.service.ValidateCode(r.Context(), code)
	if err != nil {
		if errors.Is(err, domain.ErrCodeNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "code_not_found", "discount code not found")
			return
		}
		if errors.Is(err, domain.ErrCodeExhausted) {
			httpx.WriteError(w, http.StatusBadRequest, "code_exhausted", "discount code has reached its usage limit")
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, "code_invalid", "discount code is invalid or expired")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, validateCodeResponse{
		Code:         dc.Code,
		ValuePercent: dc.ValuePercent,
		Valid:         true,
	})
}

func writeError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrPromotionNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "promotion not found")
		return
	}
	if errors.Is(err, domain.ErrCodeNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "discount code not found")
		return
	}
	if errors.Is(err, domain.ErrDuplicateCode) {
		httpx.WriteError(w, http.StatusConflict, "duplicate_code", "a discount code with that value already exists")
		return
	}
	httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
}

func parseUUIDs(raw []string) ([]uuid.UUID, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	result := make([]uuid.UUID, 0, len(raw))
	for _, s := range raw {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, nil
}
