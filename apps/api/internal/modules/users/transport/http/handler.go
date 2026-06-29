package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/users/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/users/domain"
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
		r.Get("/me", h.getProfile)
		r.Patch("/me", h.updateProfile)
		r.Get("/me/addresses", h.listAddresses)
		r.Post("/me/addresses", h.createAddress)
		r.Patch("/me/addresses/{id}", h.updateAddress)
		r.Delete("/me/addresses/{id}", h.deleteAddress)
	})

	r.Group(func(r chi.Router) {
		r.Use(requireAdmin)
		r.Get("/admin/users", h.adminListUsers)
		r.Get("/admin/users/stats", h.adminUserStats)
		r.Get("/admin/users/{id}", h.adminGetUser)
		r.Patch("/admin/users/{id}/roles", h.adminSetRoles)
	})
}

func principalFrom(r *http.Request) (authctx.Principal, bool) {
	return authctx.FromContext(r.Context())
}

func (h *Handler) getProfile(w http.ResponseWriter, r *http.Request) {
	p, ok := principalFrom(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing principal")
		return
	}

	user, err := h.service.GetProfile(r.Context(), p.UserID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toProfileResponse(user))
}

type updateProfileRequest struct {
	FullName *string `json:"full_name,omitempty"`
	Phone    *string `json:"phone,omitempty"`
}

func (h *Handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	p, ok := principalFrom(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing principal")
		return
	}

	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	user, err := h.service.UpdateProfile(r.Context(), p.UserID, application.UpdateProfileInput{
		FullName: req.FullName,
		Phone:    req.Phone,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toProfileResponse(user))
}

func (h *Handler) listAddresses(w http.ResponseWriter, r *http.Request) {
	p, ok := principalFrom(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing principal")
		return
	}

	addresses, err := h.service.ListAddresses(r.Context(), p.UserID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	resp := make([]addressResponse, 0, len(addresses))
	for _, a := range addresses {
		resp = append(resp, toAddressResponse(a))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

type addressRequest struct {
	Label         string `json:"label"`
	RecipientName string `json:"recipient_name"`
	Phone         string `json:"phone"`
	Line1         string `json:"line1"`
	Line2         string `json:"line2"`
	City          string `json:"city"`
	Region        string `json:"region"`
	PostalCode    string `json:"postal_code"`
	CountryCode   string `json:"country_code"`
	IsDefault     bool   `json:"is_default"`
}

func (h *Handler) createAddress(w http.ResponseWriter, r *http.Request) {
	p, ok := principalFrom(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing principal")
		return
	}

	var req addressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	addr, err := h.service.AddAddress(r.Context(), p.UserID, application.AddAddressInput{
		Label:         req.Label,
		RecipientName: req.RecipientName,
		Phone:         req.Phone,
		Line1:         req.Line1,
		Line2:         req.Line2,
		City:          req.City,
		Region:        req.Region,
		PostalCode:    req.PostalCode,
		CountryCode:   req.CountryCode,
		IsDefault:     req.IsDefault,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toAddressResponse(*addr))
}

func (h *Handler) updateAddress(w http.ResponseWriter, r *http.Request) {
	p, ok := principalFrom(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing principal")
		return
	}

	addressID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "address id is invalid")
		return
	}

	var req struct {
		Label         *string `json:"label,omitempty"`
		RecipientName *string `json:"recipient_name,omitempty"`
		Phone         *string `json:"phone,omitempty"`
		Line1         *string `json:"line1,omitempty"`
		Line2         *string `json:"line2,omitempty"`
		City          *string `json:"city,omitempty"`
		Region        *string `json:"region,omitempty"`
		PostalCode    *string `json:"postal_code,omitempty"`
		CountryCode   *string `json:"country_code,omitempty"`
		IsDefault     *bool   `json:"is_default,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	addr, err := h.service.UpdateAddress(r.Context(), p.UserID, addressID, application.UpdateAddressInput{
		Label:         req.Label,
		RecipientName: req.RecipientName,
		Phone:         req.Phone,
		Line1:         req.Line1,
		Line2:         req.Line2,
		City:          req.City,
		Region:        req.Region,
		PostalCode:    req.PostalCode,
		CountryCode:   req.CountryCode,
		IsDefault:     req.IsDefault,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toAddressResponse(*addr))
}

func (h *Handler) deleteAddress(w http.ResponseWriter, r *http.Request) {
	p, ok := principalFrom(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing principal")
		return
	}

	addressID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "address id is invalid")
		return
	}

	if err := h.service.DeleteAddress(r.Context(), p.UserID, addressID); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) adminListUsers(w http.ResponseWriter, r *http.Request) {
	filter := application.ListUsersFilter{
		Search:   r.URL.Query().Get("search"),
		Page:     1,
		PageSize: 20,
	}
	if page, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && page > 0 {
		filter.Page = page
	}
	if pageSize, err := strconv.Atoi(r.URL.Query().Get("page_size")); err == nil && pageSize > 0 {
		filter.PageSize = pageSize
	}

	result, err := h.service.AdminListUsers(r.Context(), filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toAdminUserListResponse(result, filter.Page, filter.PageSize))
}

func (h *Handler) adminGetUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "user id is invalid")
		return
	}

	view, err := h.service.AdminGetUser(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toAdminUserResponse(view))
}

type countBreakdownResponse struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type dailyUserCountResponse struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type userStatsResponse struct {
	TotalUsers         int                      `json:"total_users"`
	New24h             int                      `json:"new_24h"`
	New7d              int                      `json:"new_7d"`
	New30d             int                      `json:"new_30d"`
	RoleBreakdown      []countBreakdownResponse `json:"role_breakdown"`
	ByCountry          []countBreakdownResponse `json:"by_country"`
	DailyRegistrations []dailyUserCountResponse `json:"daily_registrations"`
}

func toCountBreakdownResponses(items []application.CountBreakdown) []countBreakdownResponse {
	resp := make([]countBreakdownResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, countBreakdownResponse{Label: item.Label, Count: item.Count})
	}
	return resp
}

func (h *Handler) adminUserStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.UserStats(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	resp := userStatsResponse{
		TotalUsers:         stats.TotalUsers,
		New24h:             stats.New24h,
		New7d:              stats.New7d,
		New30d:             stats.New30d,
		RoleBreakdown:      toCountBreakdownResponses(stats.RoleBreakdown),
		ByCountry:          toCountBreakdownResponses(stats.ByCountry),
		DailyRegistrations: make([]dailyUserCountResponse, 0, len(stats.DailyRegistrations)),
	}
	for _, d := range stats.DailyRegistrations {
		resp.DailyRegistrations = append(resp.DailyRegistrations, dailyUserCountResponse{
			Date:  d.Date.Format("2006-01-02"),
			Count: d.Count,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

type setRolesRequest struct {
	Roles []string `json:"roles"`
}

func (h *Handler) adminSetRoles(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "user id is invalid")
		return
	}

	var req setRolesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	roles := make([]domain.Role, 0, len(req.Roles))
	for _, role := range req.Roles {
		roles = append(roles, domain.Role(role))
	}

	user, err := h.service.AdminSetRoles(r.Context(), id, roles)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	view, err := h.service.AdminGetUser(r.Context(), user.ID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toAdminUserResponse(view))
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		httpx.WriteError(w, http.StatusNotFound, "user_not_found", "user not found")
	case errors.Is(err, domain.ErrAddressNotFound):
		httpx.WriteError(w, http.StatusNotFound, "address_not_found", "address not found")
	case errors.As(err, new(domain.ValidationError)):
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", err.Error())
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
