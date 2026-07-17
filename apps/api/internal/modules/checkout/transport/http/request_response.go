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

type placeOrderRequest struct {
	Contact          contactRequest `json:"contact"`
	ShippingAddress  addressRequest `json:"shipping_address"`
	BillingAddress   addressRequest `json:"billing_address"`
	DeliveryMethod   string         `json:"delivery_method"`
	DeliveryOfficeID string         `json:"delivery_office_id,omitempty"`
	PaymentMethod    string         `json:"payment_method"`
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
		DiscountCode:     req.DiscountCode,
	}
}

// refundRequest is the admin refund submission. AmountMinor must be positive
// and no greater than the order's remaining refundable amount; the admin UI
// sends captured − already-refunded for a full refund.
type refundRequest struct {
	AmountMinor int64  `json:"amount_minor"`
	Reason      string `json:"reason,omitempty"`
}

// paymentInitiationResponse is returned for online-card orders: the order is
// created as pending_payment and the client mounts the Revolut widget with the
// token, then polls the order until the payment webhook settles it.
type paymentInitiationResponse struct {
	OrderID           string        `json:"order_id"`
	OrderNumber       string        `json:"order_number"`
	RevolutOrderID    string        `json:"revolut_order_id"`
	RevolutOrderToken string        `json:"revolut_order_token"`
	Amount            moneyResponse `json:"amount"`
	PaymentMethod     string        `json:"payment_method"`
	Status            string        `json:"status"`
	// RequiresPayment always true here — lets the client discriminate this
	// response from a fully placed (pay-on-delivery) order in the same 201.
	RequiresPayment bool `json:"requires_payment"`
}

func toPaymentInitiationResponse(p application.PaymentInitiation) paymentInitiationResponse {
	return paymentInitiationResponse{
		OrderID:           p.OrderID.String(),
		OrderNumber:       p.OrderNumber,
		RevolutOrderID:    p.RevolutOrderID,
		RevolutOrderToken: p.RevolutOrderToken,
		Amount:            moneyResponse{AmountMinor: p.Amount.AmountMinor, Currency: p.Amount.Currency},
		PaymentMethod:     p.PaymentMethod,
		Status:            p.Status,
		RequiresPayment:   true,
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
