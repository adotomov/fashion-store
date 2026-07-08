package http

import (
	"github.com/adotomov/fashion-store/apps/api/internal/modules/checkout/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/checkout/domain"
)

const timeFormat = "2006-01-02T15:04:05Z07:00"

type moneyResponse struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

type deliveryMethodResponse struct {
	Code string        `json:"code"`
	Name string        `json:"name"`
	Fee  moneyResponse `json:"fee"`
	// PaymentMethods lists the payment methods compatible with this delivery
	// method, so the storefront only offers valid combinations at checkout.
	PaymentMethods []string `json:"payment_methods"`
}

func toDeliveryMethodResponse(m domain.DeliveryMethod) deliveryMethodResponse {
	return deliveryMethodResponse{
		Code: m.Code, Name: m.Name,
		Fee:            moneyResponse{AmountMinor: m.Fee.AmountMinor, Currency: m.Fee.Currency},
		PaymentMethods: domain.PaymentMethodsFor(m.Code),
	}
}

type contactRequest struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

type addressRequest struct {
	RecipientName string `json:"recipient_name"`
	Phone         string `json:"phone"`
	Line1         string `json:"line1"`
	Line2         string `json:"line2"`
	City          string `json:"city"`
	Region        string `json:"region"`
	PostalCode    string `json:"postal_code"`
	CountryCode   string `json:"country_code"`
}

func (a addressRequest) toInput() application.AddressInput {
	return application.AddressInput{
		RecipientName: a.RecipientName, Phone: a.Phone, Line1: a.Line1, Line2: a.Line2,
		City: a.City, Region: a.Region, PostalCode: a.PostalCode, CountryCode: a.CountryCode,
	}
}

type cardRequest struct {
	Number   string `json:"number"`
	ExpMonth int    `json:"exp_month"`
	ExpYear  int    `json:"exp_year"`
	CVV      string `json:"cvv"`
}

type placeOrderRequest struct {
	Contact          contactRequest `json:"contact"`
	ShippingAddress  addressRequest `json:"shipping_address"`
	BillingAddress   addressRequest `json:"billing_address"`
	DeliveryMethod   string         `json:"delivery_method"`
	DeliveryOfficeID string         `json:"delivery_office_id,omitempty"`
	PaymentMethod    string         `json:"payment_method"`
	Card             cardRequest    `json:"card"`
	DiscountCode     string         `json:"discount_code,omitempty"`
}

func (req placeOrderRequest) toInput() application.PlaceOrderInput {
	billing := req.BillingAddress
	if billing == (addressRequest{}) {
		billing = req.ShippingAddress
	}
	return application.PlaceOrderInput{
		Contact: application.ContactInput{
			FullName: req.Contact.FullName, Email: req.Contact.Email, Phone: req.Contact.Phone,
		},
		ShippingAddress:  req.ShippingAddress.toInput(),
		BillingAddress:   billing.toInput(),
		DeliveryMethod:   req.DeliveryMethod,
		DeliveryOfficeID: req.DeliveryOfficeID,
		PaymentMethod:    req.PaymentMethod,
		Card: application.CardInput{
			Number: req.Card.Number, ExpMonth: req.Card.ExpMonth, ExpYear: req.Card.ExpYear, CVV: req.Card.CVV,
		},
		DiscountCode: req.DiscountCode,
	}
}

type orderItemResponse struct {
	ProductName  string        `json:"product_name"`
	VariantLabel string        `json:"variant_label,omitempty"`
	Quantity     int           `json:"quantity"`
	UnitPrice    moneyResponse `json:"unit_price"`
}

type orderResultResponse struct {
	ID             string              `json:"id"`
	OrderNumber    string              `json:"order_number"`
	Status         string              `json:"status"`
	Total          moneyResponse       `json:"total"`
	DeliveryMethod string              `json:"delivery_method"`
	DeliveryFee    moneyResponse       `json:"delivery_fee"`
	PaymentMethod  string              `json:"payment_method"`
	PlacedAt       string              `json:"placed_at"`
	DiscountCode   *string             `json:"discount_code,omitempty"`
	DiscountAmount *moneyResponse      `json:"discount_amount,omitempty"`
	Items          []orderItemResponse `json:"items"`
}

func toOrderResultResponse(o application.OrderResult) orderResultResponse {
	resp := orderResultResponse{
		ID:             o.ID.String(),
		OrderNumber:    o.OrderNumber,
		Status:         o.Status,
		Total:          moneyResponse{AmountMinor: o.Total.AmountMinor, Currency: o.Total.Currency},
		DeliveryMethod: o.DeliveryMethod,
		DeliveryFee:    moneyResponse{AmountMinor: o.DeliveryFee.AmountMinor, Currency: o.DeliveryFee.Currency},
		PaymentMethod:  o.PaymentMethod,
		PlacedAt:       o.PlacedAt.Format(timeFormat),
		DiscountCode:   o.DiscountCode,
		Items:          make([]orderItemResponse, 0, len(o.Items)),
	}
	if o.DiscountAmount != nil {
		dm := moneyResponse{AmountMinor: o.DiscountAmount.AmountMinor, Currency: o.DiscountAmount.Currency}
		resp.DiscountAmount = &dm
	}
	for _, item := range o.Items {
		resp.Items = append(resp.Items, orderItemResponse{
			ProductName:  item.ProductName,
			VariantLabel: item.VariantLabel,
			Quantity:     item.Quantity,
			UnitPrice:    moneyResponse{AmountMinor: item.UnitPrice.AmountMinor, Currency: item.UnitPrice.Currency},
		})
	}
	return resp
}
