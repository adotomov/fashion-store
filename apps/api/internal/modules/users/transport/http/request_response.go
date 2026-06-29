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
	Line1         string `json:"line1"`
	Line2         string `json:"line2"`
	City          string `json:"city"`
	Region        string `json:"region"`
	PostalCode    string `json:"postal_code"`
	CountryCode   string `json:"country_code"`
	IsDefault     bool   `json:"is_default"`
}

func toAddressResponse(a domain.Address) addressResponse {
	return addressResponse{
		ID:            a.ID.String(),
		Label:         a.Label,
		RecipientName: a.RecipientName,
		Phone:         a.Phone,
		Line1:         a.Line1,
		Line2:         a.Line2,
		City:          a.City,
		Region:        a.Region,
		PostalCode:    a.PostalCode,
		CountryCode:   a.CountryCode,
		IsDefault:     a.IsDefault,
	}
}
