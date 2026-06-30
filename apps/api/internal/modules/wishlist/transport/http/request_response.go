package http

import (
	"github.com/adotomov/fashion-store/apps/api/internal/modules/wishlist/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

type moneyResponse struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

func toMoneyResponse(m money.Money) moneyResponse {
	return moneyResponse{AmountMinor: m.AmountMinor, Currency: m.Currency}
}

type itemResponse struct {
	ID             string         `json:"id"`
	ProductID      string         `json:"product_id"`
	ProductName    string         `json:"product_name"`
	ProductSlug    string         `json:"product_slug"`
	ImageURL       *string        `json:"image_url,omitempty"`
	BasePrice      moneyResponse  `json:"base_price"`
	CompareAtPrice *moneyResponse `json:"compare_at_price,omitempty"`
	InStock        bool           `json:"in_stock"`
	Sizes          []string       `json:"sizes"`
	CreatedAt      string         `json:"created_at"`
}

const timeFormat = "2006-01-02T15:04:05Z07:00"

func toItemResponse(item domain.Item) itemResponse {
	resp := itemResponse{
		ID:          item.ID.String(),
		ProductID:   item.ProductID.String(),
		ProductName: item.ProductName,
		ProductSlug: item.ProductSlug,
		BasePrice:   toMoneyResponse(item.BasePrice),
		InStock:     item.InStock,
		Sizes:       item.Sizes,
		CreatedAt:   item.CreatedAt.Format(timeFormat),
	}
	if item.Sizes == nil {
		resp.Sizes = []string{}
	}
	if item.CompareAtPrice != nil {
		compareAtPrice := toMoneyResponse(*item.CompareAtPrice)
		resp.CompareAtPrice = &compareAtPrice
	}
	if item.ImageMediaID != nil {
		url := "/api/v1/storefront/media/" + item.ImageMediaID.String() + "/file"
		resp.ImageURL = &url
	}
	return resp
}
