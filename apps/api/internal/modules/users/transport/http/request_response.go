package http

import (
	"time"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/users/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/users/domain"
)

type profileResponse struct {
	ID       string   `json:"id"`
	Email    string   `json:"email"`
	FullName string   `json:"full_name"`
	Phone    string   `json:"phone"`
	Roles    []string `json:"roles"`
	// Locale is the account's preferred language, so the storefront can display
	// in it (signed-in account language wins over geo detection).
	Locale string `json:"locale"`
}

func toProfileResponse(u *domain.User) profileResponse {
	roles := make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		roles = append(roles, string(r))
	}
	return profileResponse{
		ID:       u.ID.String(),
		Email:    u.Email,
		FullName: u.FullName,
		Phone:    u.Phone,
		Roles:    roles,
		Locale:   u.Locale,
	}
}

type adminUserResponse struct {
	ID         string   `json:"id"`
	Email      string   `json:"email"`
	FullName   string   `json:"full_name"`
	Phone      string   `json:"phone"`
	Roles      []string `json:"roles"`
	OrderCount int      `json:"order_count"`
	CreatedAt  string   `json:"created_at"`
}

func toAdminUserResponse(view application.AdminUserView) adminUserResponse {
	u := view.User
	roles := make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		roles = append(roles, string(r))
	}
	return adminUserResponse{
		ID:         u.ID.String(),
		Email:      u.Email,
		FullName:   u.FullName,
		Phone:      u.Phone,
		Roles:      roles,
		OrderCount: view.OrderCount,
		CreatedAt:  u.CreatedAt.Format(time.RFC3339),
	}
}

type adminUserListResponse struct {
	Users    []adminUserResponse `json:"users"`
	Total    int                 `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

func toAdminUserListResponse(result application.ListUsersResult, page, pageSize int) adminUserListResponse {
	resp := make([]adminUserResponse, 0, len(result.Users))
	for _, view := range result.Users {
		resp = append(resp, toAdminUserResponse(view))
	}
	return adminUserListResponse{Users: resp, Total: result.Total, Page: page, PageSize: pageSize}
}

type addressResponse struct {
	ID            string `json:"id"`
	Label         string `json:"label"`
	RecipientName string `json:"recipient_name"`
	Phone         string `json:"phone"`
	CountryCode   string `json:"country_code"`
	CountryID     int64  `json:"country_id"`
	SiteID        int64  `json:"site_id"`
	City          string `json:"city"`
	PostCode      string `json:"post_code"`
	ComplexID     int64  `json:"complex_id"`
	ComplexName   string `json:"complex_name"`
	StreetID      int64  `json:"street_id"`
	StreetName    string `json:"street_name"`
	StreetNo      string `json:"street_no"`
	BlockNo       string `json:"block_no"`
	EntranceNo    string `json:"entrance_no"`
	FloorNo       string `json:"floor_no"`
	ApartmentNo   string `json:"apartment_no"`
	IsDefault     bool   `json:"is_default"`
}

func toAddressResponse(a domain.Address) addressResponse {
	return addressResponse{
		ID:            a.ID.String(),
		Label:         a.Label,
		RecipientName: a.RecipientName,
		Phone:         a.Phone,
		CountryCode:   a.CountryCode,
		CountryID:     a.CountryID,
		SiteID:        a.SiteID,
		City:          a.City,
		PostCode:      a.PostCode,
		ComplexID:     a.ComplexID,
		ComplexName:   a.ComplexName,
		StreetID:      a.StreetID,
		StreetName:    a.StreetName,
		StreetNo:      a.StreetNo,
		BlockNo:       a.BlockNo,
		EntranceNo:    a.EntranceNo,
		FloorNo:       a.FloorNo,
		ApartmentNo:   a.ApartmentNo,
		IsDefault:     a.IsDefault,
	}
}
