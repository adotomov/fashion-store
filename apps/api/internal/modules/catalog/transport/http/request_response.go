package http

import "github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"

type catalogResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

const timeFormat = "2006-01-02T15:04:05Z07:00"

func toCatalogResponse(c domain.Catalog) catalogResponse {
	return catalogResponse{
		ID:          c.ID.String(),
		Name:        c.Name,
		Slug:        c.Slug,
		Description: c.Description,
		Status:      string(c.Status),
		CreatedAt:   c.CreatedAt.Format(timeFormat),
		UpdatedAt:   c.UpdatedAt.Format(timeFormat),
	}
}
