package http

import (
	"github.com/adotomov/fashion-store/apps/api/internal/modules/orders/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/orders/domain"
)

const timeFormat = "2006-01-02T15:04:05Z07:00"

type moneyResponse struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

type orderItemResponse struct {
	ID           string        `json:"id"`
	ProductName  string        `json:"product_name"`
	VariantLabel string        `json:"variant_label,omitempty"`
	Quantity     int           `json:"quantity"`
	UnitPrice    moneyResponse `json:"unit_price"`
}

type orderAddressResponse struct {
	RecipientName string `json:"recipient_name"`
	Phone         string `json:"phone"`
	Line1         string `json:"line1"`
	Line2         string `json:"line2"`
	City          string `json:"city"`
	Region        string `json:"region"`
	PostalCode    string `json:"postal_code"`
	CountryCode   string `json:"country_code"`
}

func toOrderAddressResponse(a domain.OrderAddress) orderAddressResponse {
	return orderAddressResponse{
		RecipientName: a.RecipientName, Phone: a.Phone, Line1: a.Line1, Line2: a.Line2,
		City: a.City, Region: a.Region, PostalCode: a.PostalCode, CountryCode: a.CountryCode,
	}
}

type orderPaymentResponse struct {
	Provider          string        `json:"provider"`
	ProviderReference string        `json:"provider_reference,omitempty"`
	Status            string        `json:"status"`
	Amount            moneyResponse `json:"amount"`
	Captured          moneyResponse `json:"captured"`
	Refunded          moneyResponse `json:"refunded"`
}

type paymentTransactionResponse struct {
	ID                string        `json:"id"`
	Type              string        `json:"type"`
	Status            string        `json:"status,omitempty"`
	Provider          string        `json:"provider"`
	ProviderReference string        `json:"provider_reference,omitempty"`
	Amount            moneyResponse `json:"amount"`
	CreatedAt         string        `json:"created_at"`
}

func toPaymentTransactionResponses(txns []domain.PaymentTransaction) []paymentTransactionResponse {
	resp := make([]paymentTransactionResponse, 0, len(txns))
	for _, t := range txns {
		resp = append(resp, paymentTransactionResponse{
			ID:                t.ID.String(),
			Type:              t.Type,
			Status:            t.Status,
			Provider:          t.Provider,
			ProviderReference: t.ProviderReference,
			Amount:            moneyResponse{AmountMinor: t.Amount.AmountMinor, Currency: t.Amount.Currency},
			CreatedAt:         t.CreatedAt.Format(timeFormat),
		})
	}
	return resp
}

type orderResponse struct {
	ID          string              `json:"id"`
	OrderNumber string              `json:"order_number"`
	Status      string              `json:"status"`
	Total       moneyResponse       `json:"total"`
	PlacedAt    string              `json:"placed_at"`
	Items       []orderItemResponse `json:"items"`

	ContactName  string `json:"contact_name,omitempty"`
	ContactEmail string `json:"contact_email,omitempty"`
	ContactPhone string `json:"contact_phone,omitempty"`

	ShippingAddress orderAddressResponse `json:"shipping_address"`
	BillingAddress  orderAddressResponse `json:"billing_address"`

	DeliveryMethod string                `json:"delivery_method"`
	DeliveryFee    moneyResponse         `json:"delivery_fee"`
	PaymentMethod  string                `json:"payment_method"`
	Payment        *orderPaymentResponse `json:"payment,omitempty"`

	Carrier         string `json:"carrier,omitempty"`
	TrackingNumber  string `json:"tracking_number,omitempty"`
	ShipmentStatus  string `json:"shipment_status,omitempty"`
	ViewedByAdminAt string `json:"viewed_by_admin_at,omitempty"`
}

func toOrderResponse(o domain.Order) orderResponse {
	resp := orderResponse{
		ID:           o.ID.String(),
		OrderNumber:  o.OrderNumber,
		Status:       string(o.Status),
		Total:        moneyResponse{AmountMinor: o.Total.AmountMinor, Currency: o.Total.Currency},
		PlacedAt:     o.PlacedAt.Format(timeFormat),
		Items:        make([]orderItemResponse, 0, len(o.Items)),
		ContactName:  o.ContactName,
		ContactEmail: o.ContactEmail,
		ContactPhone: o.ContactPhone,

		ShippingAddress: toOrderAddressResponse(o.ShippingAddress),
		BillingAddress:  toOrderAddressResponse(o.BillingAddress),

		DeliveryMethod: o.DeliveryMethod,
		DeliveryFee:    moneyResponse{AmountMinor: o.DeliveryFee.AmountMinor, Currency: o.DeliveryFee.Currency},
		PaymentMethod:  o.PaymentMethod,
	}
	if o.Payment != nil {
		resp.Payment = &orderPaymentResponse{
			Provider:          o.Payment.Provider,
			ProviderReference: o.Payment.ProviderReference,
			Status:            o.Payment.Status,
			Amount:            moneyResponse{AmountMinor: o.Payment.Amount.AmountMinor, Currency: o.Payment.Amount.Currency},
			Captured:          moneyResponse{AmountMinor: o.Payment.CapturedMinor, Currency: o.Payment.Amount.Currency},
			Refunded:          moneyResponse{AmountMinor: o.Payment.RefundedMinor, Currency: o.Payment.Amount.Currency},
		}
	}
	if o.Carrier != nil {
		resp.Carrier = *o.Carrier
	}
	if o.TrackingNumber != nil {
		resp.TrackingNumber = *o.TrackingNumber
	}
	if o.ShipmentStatus != nil {
		resp.ShipmentStatus = *o.ShipmentStatus
	}
	if o.ViewedByAdminAt != nil {
		resp.ViewedByAdminAt = o.ViewedByAdminAt.Format(timeFormat)
	}
	for _, item := range o.Items {
		resp.Items = append(resp.Items, orderItemResponse{
			ID:           item.ID.String(),
			ProductName:  item.ProductName,
			VariantLabel: item.VariantLabel,
			Quantity:     item.Quantity,
			UnitPrice:    moneyResponse{AmountMinor: item.UnitPrice.AmountMinor, Currency: item.UnitPrice.Currency},
		})
	}
	return resp
}

type countBreakdownResponse struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type dailyOrderCountResponse struct {
	Date    string        `json:"date"`
	Count   int           `json:"count"`
	Revenue moneyResponse `json:"revenue"`
}

type orderStatsResponse struct {
	OrderCount       int                       `json:"order_count"`
	Revenue          moneyResponse             `json:"revenue"`
	AvgOrderValue    moneyResponse             `json:"avg_order_value"`
	StatusBreakdown  []countBreakdownResponse  `json:"status_breakdown"`
	ByCity           []countBreakdownResponse  `json:"by_city"`
	ByCountry        []countBreakdownResponse  `json:"by_country"`
	ByDeliveryMethod []countBreakdownResponse  `json:"by_delivery_method"`
	DailyCounts      []dailyOrderCountResponse `json:"daily_counts"`
}

func toCountBreakdownResponses(items []application.CountBreakdown) []countBreakdownResponse {
	resp := make([]countBreakdownResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, countBreakdownResponse{Label: item.Label, Count: item.Count})
	}
	return resp
}

func toOrderStatsResponse(s application.OrderStats) orderStatsResponse {
	resp := orderStatsResponse{
		OrderCount:       s.OrderCount,
		Revenue:          moneyResponse{AmountMinor: s.Revenue.AmountMinor, Currency: s.Revenue.Currency},
		AvgOrderValue:    moneyResponse{AmountMinor: s.AvgOrderValue.AmountMinor, Currency: s.AvgOrderValue.Currency},
		StatusBreakdown:  toCountBreakdownResponses(s.StatusBreakdown),
		ByCity:           toCountBreakdownResponses(s.ByCity),
		ByCountry:        toCountBreakdownResponses(s.ByCountry),
		ByDeliveryMethod: toCountBreakdownResponses(s.ByDeliveryMethod),
		DailyCounts:      make([]dailyOrderCountResponse, 0, len(s.DailyCounts)),
	}
	for _, d := range s.DailyCounts {
		resp.DailyCounts = append(resp.DailyCounts, dailyOrderCountResponse{
			Date:    d.Date.Format("2006-01-02"),
			Count:   d.Count,
			Revenue: moneyResponse{AmountMinor: d.Revenue.AmountMinor, Currency: d.Revenue.Currency},
		})
	}
	return resp
}
