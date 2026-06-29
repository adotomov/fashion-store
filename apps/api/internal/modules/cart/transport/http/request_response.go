package http

import (
	"github.com/adotomov/fashion-store/apps/api/internal/modules/cart/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

type moneyResponse struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

func toMoneyResponse(m money.Money) moneyResponse {
	return moneyResponse{AmountMinor: m.AmountMinor, Currency: m.Currency}
}

type cartItemResponse struct {
	ID                string        `json:"id"`
	VariantID         string        `json:"variant_id"`
	ProductID         string        `json:"product_id"`
	ProductName       string        `json:"product_name"`
	ProductSlug       string        `json:"product_slug"`
	VariantLabel      string        `json:"variant_label,omitempty"`
	ImageURL          *string       `json:"image_url,omitempty"`
	UnitPrice         moneyResponse `json:"unit_price"`
	LineTotal         moneyResponse `json:"line_total"`
	Quantity          int           `json:"quantity"`
	AvailableQuantity int           `json:"available_quantity"`
}

type cartResponse struct {
	ID         string             `json:"id"`
	GuestToken *string            `json:"guest_token,omitempty"`
	Items      []cartItemResponse `json:"items"`
	Subtotal   moneyResponse      `json:"subtotal"`
	ItemCount  int                `json:"item_count"`
}

func toCartItemResponse(item domain.CartItem) cartItemResponse {
	resp := cartItemResponse{
		ID:                item.ID.String(),
		VariantID:         item.VariantID.String(),
		ProductID:         item.ProductID.String(),
		ProductName:       item.ProductName,
		ProductSlug:       item.ProductSlug,
		VariantLabel:      item.VariantLabel,
		UnitPrice:         toMoneyResponse(item.UnitPrice),
		Quantity:          item.Quantity,
		AvailableQuantity: item.AvailableQuantity,
		LineTotal: toMoneyResponse(money.Money{
			AmountMinor: item.UnitPrice.AmountMinor * int64(item.Quantity),
			Currency:    item.UnitPrice.Currency,
		}),
	}
	if item.ImageMediaID != nil {
		url := "/api/v1/storefront/media/" + item.ImageMediaID.String() + "/file"
		resp.ImageURL = &url
	}
	return resp
}

func toCartResponse(cart domain.Cart) cartResponse {
	resp := cartResponse{
		ID:    cart.ID.String(),
		Items: make([]cartItemResponse, 0, len(cart.Items)),
	}
	if cart.GuestToken != nil {
		token := cart.GuestToken.String()
		resp.GuestToken = &token
	}

	var subtotal int64
	currency := "EUR"
	for _, item := range cart.Items {
		resp.Items = append(resp.Items, toCartItemResponse(item))
		subtotal += item.UnitPrice.AmountMinor * int64(item.Quantity)
		resp.ItemCount += item.Quantity
		currency = item.UnitPrice.Currency
	}
	resp.Subtotal = toMoneyResponse(money.Money{AmountMinor: subtotal, Currency: currency})
	return resp
}
