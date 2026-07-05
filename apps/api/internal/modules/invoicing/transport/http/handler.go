package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/invoicing/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/invoicing/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/authctx"
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
		r.Get("/admin/invoices", h.list)
		r.Get("/admin/invoices/export", h.exportCSV)
		r.Get("/admin/invoices/{id}", h.get)
		r.Post("/admin/invoices/generate/{orderID}", h.generate)
		r.Post("/admin/invoices/{id}/storno", h.storno)
		r.Get("/admin/invoices/{id}/view", h.viewHTML)
		r.Get("/admin/invoice-settings", h.getSettings)
		r.Put("/admin/invoice-settings", h.saveSettings)
		r.Get("/admin/invoice-couriers", h.listCouriers)
		r.Post("/admin/invoice-couriers", h.createCourier)
		r.Put("/admin/invoice-couriers/{id}", h.updateCourier)
		r.Delete("/admin/invoice-couriers/{id}", h.deleteCourier)
	})
}

// ── Invoice endpoints ────────────────────────────────────────────────────────

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := application.ListFilter{
		Search: q.Get("q"),
		Limit:  50,
	}
	if v := q.Get("type"); v != "" {
		filter.DocumentType = &v
	}
	if v := q.Get("payment_method"); v != "" {
		filter.PaymentMethod = &v
	}
	if v := q.Get("from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.From = &t
		}
	}
	if v := q.Get("to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			end := t.Add(24*time.Hour - time.Second)
			filter.To = &end
		}
	}

	invoices, err := h.service.List(r.Context(), filter)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	resp := make([]invoiceListItem, 0, len(invoices))
	for _, inv := range invoices {
		resp = append(resp, toListItem(inv))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid invoice id")
		return
	}
	inv, err := h.service.FindByID(r.Context(), id)
	if errors.Is(err, domain.ErrInvoiceNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "invoice not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "load_failed", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toDetailResponse(*inv))
}

func (h *Handler) generate(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(chi.URLParam(r, "orderID"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid order id")
		return
	}
	actor := actorFromContext(r)
	inv, err := h.service.GenerateForOrder(r.Context(), orderID, actor)
	if errors.Is(err, domain.ErrSettingsIncomplete) {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "settings_incomplete", err.Error())
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "generate_failed", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toListItem(*inv))
}

func (h *Handler) storno(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid invoice id")
		return
	}
	actor := actorFromContext(r)
	inv, err := h.service.GenerateStorno(r.Context(), id, actor)
	if errors.Is(err, domain.ErrInvoiceNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "invoice not found")
		return
	}
	if errors.Is(err, domain.ErrCannotStornoAStorno) {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid_operation", err.Error())
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "storno_failed", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toListItem(*inv))
}

func (h *Handler) exportCSV(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	fromStr := q.Get("from")
	toStr := q.Get("to")
	if fromStr == "" || toStr == "" {
		httpx.WriteError(w, http.StatusBadRequest, "missing_params", "from and to query params are required")
		return
	}
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_from", "from must be YYYY-MM-DD")
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_to", "to must be YYYY-MM-DD")
		return
	}
	to = to.Add(24*time.Hour - time.Second)

	actor := actorFromContext(r)
	data, err := h.service.ExportCSV(r.Context(), from, to, actor)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "export_failed", err.Error())
		return
	}

	filename := fmt.Sprintf("invoices_%s_%s.csv", fromStr, toStr)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	// UTF-8 BOM for Excel compatibility
	w.Write([]byte{0xEF, 0xBB, 0xBF})
	w.Write(data)
}

func (h *Handler) viewHTML(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid invoice id", http.StatusBadRequest)
		return
	}
	inv, err := h.service.FindByID(r.Context(), id)
	if errors.Is(err, domain.ErrInvoiceNotFound) {
		http.Error(w, "invoice not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "could not load invoice", http.StatusInternalServerError)
		return
	}

	go h.service.LogPDFView(r.Context(), inv, actorFromContext(r))

	html, err := renderInvoiceHTML(inv)
	if err != nil {
		http.Error(w, "could not render invoice", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(html)
}

// ── Settings endpoints ────────────────────────────────────────────────────────

func (h *Handler) getSettings(w http.ResponseWriter, r *http.Request) {
	s, err := h.service.GetSettings(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "load_failed", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toSettingsResponse(s))
}

func (h *Handler) saveSettings(w http.ResponseWriter, r *http.Request) {
	var req invoiceSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}
	settings := domain.InvoiceSettings{
		CompanyName:             req.CompanyName,
		CompanyLegalType:        req.CompanyLegalType,
		CompanyEIK:              req.CompanyEIK,
		CompanyAddressStreet:     req.CompanyAddressStreet,
		CompanyAddressCity:       req.CompanyAddressCity,
		CompanyAddressPostalCode: req.CompanyAddressPostalCode,
		CompanyAddressCountry:    req.CompanyAddressCountry,
		CompanyEmail:            req.CompanyEmail,
		CompanyPhone:            req.CompanyPhone,
		NRAStoreNumber:          req.NRAStoreNumber,
		VATNumber:               req.VATNumber,
		VATRate:                 20.0,
	}
	if err := h.service.SaveSettings(r.Context(), settings); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "save_failed", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toSettingsResponse(settings))
}

// ── Courier endpoints ─────────────────────────────────────────────────────────

func (h *Handler) listCouriers(w http.ResponseWriter, r *http.Request) {
	couriers, err := h.service.ListCouriers(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "load_failed", err.Error())
		return
	}
	resp := make([]courierResponse, 0, len(couriers))
	for _, c := range couriers {
		resp = append(resp, toCourierResponse(c))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) createCourier(w http.ResponseWriter, r *http.Request) {
	var req courierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}
	courier, err := h.service.CreateCourier(r.Context(), domain.Courier{
		Name:       req.Name,
		Identifier: req.Identifier,
		IsActive:   req.IsActive,
		SortOrder:  req.SortOrder,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toCourierResponse(*courier))
}

func (h *Handler) updateCourier(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid courier id")
		return
	}
	var req courierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}
	courier, err := h.service.UpdateCourier(r.Context(), id, domain.Courier{
		Name:       req.Name,
		Identifier: req.Identifier,
		IsActive:   req.IsActive,
		SortOrder:  req.SortOrder,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toCourierResponse(*courier))
}

func (h *Handler) deleteCourier(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "invalid courier id")
		return
	}
	if err := h.service.DeleteCourier(r.Context(), id); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Request/response types ────────────────────────────────────────────────────

type invoiceListItem struct {
	ID            string  `json:"id"`
	InvoiceNumber string  `json:"invoice_number"`
	DocumentType  string  `json:"document_type"`
	OrderID       string  `json:"order_id"`
	OrderNumber   string  `json:"order_number"`
	PaymentMethod string  `json:"payment_method"`
	RecipientName string  `json:"recipient_name"`
	TotalInclVAT  float64 `json:"total_incl_vat"`
	Currency      string  `json:"currency"`
	CreatedAt     string  `json:"created_at"`

	StornoOfInvoiceID *string `json:"storno_of_invoice_id,omitempty"`
}

type invoiceDetailResponse struct {
	invoiceListItem
	RecipientAddress      string            `json:"recipient_address"`
	RecipientEmail        string            `json:"recipient_email"`
	CompanyName           string            `json:"company_name"`
	CompanyLegalType      string            `json:"company_legal_type"`
	CompanyEIK            string            `json:"company_eik"`
	NRAStoreNumber        string            `json:"nra_store_number"`
	CardProvider          *string           `json:"card_provider,omitempty"`
	CardProviderReference *string           `json:"card_provider_reference,omitempty"`
	CourierName           *string           `json:"courier_name,omitempty"`
	CourierIdentifier     *string           `json:"courier_identifier,omitempty"`
	SubtotalExclVAT       float64           `json:"subtotal_excl_vat"`
	VATAmount             float64           `json:"vat_amount"`
	VATRate               float64           `json:"vat_rate"`
	DeliveryFee           float64           `json:"delivery_fee"`
	DiscountAmount        *float64          `json:"discount_amount,omitempty"`
	LineItems             []lineItemResponse `json:"line_items"`
}

type lineItemResponse struct {
	ProductName          string  `json:"product_name"`
	VariantLabel         string  `json:"variant_label"`
	NKSCode              string  `json:"nks_code"`
	Quantity             int     `json:"quantity"`
	UnitPriceInclVAT     float64 `json:"unit_price_incl_vat"`
	UnitPriceExclVAT     float64 `json:"unit_price_excl_vat"`
	VATPerUnit           float64 `json:"vat_per_unit"`
	LineTotalInclVAT     float64 `json:"line_total_incl_vat"`
	LineTotalExclVAT     float64 `json:"line_total_excl_vat"`
	LineVATAmount        float64 `json:"line_vat_amount"`
}

type invoiceSettingsRequest struct {
	CompanyName             string `json:"company_name"`
	CompanyLegalType        string `json:"company_legal_type"`
	CompanyEIK              string `json:"company_eik"`
	CompanyAddressStreet     string `json:"company_address_street"`
	CompanyAddressCity       string `json:"company_address_city"`
	CompanyAddressPostalCode string `json:"company_address_postal_code"`
	CompanyAddressCountry    string `json:"company_address_country"`
	CompanyEmail            string `json:"company_email"`
	CompanyPhone            string `json:"company_phone"`
	NRAStoreNumber          string `json:"nra_store_number"`
	VATNumber               string `json:"vat_number"`
}

type invoiceSettingsResponse struct {
	CompanyName             string  `json:"company_name"`
	CompanyLegalType        string  `json:"company_legal_type"`
	CompanyEIK              string  `json:"company_eik"`
	CompanyAddressStreet     string  `json:"company_address_street"`
	CompanyAddressCity       string  `json:"company_address_city"`
	CompanyAddressPostalCode string  `json:"company_address_postal_code"`
	CompanyAddressCountry    string  `json:"company_address_country"`
	CompanyEmail            string  `json:"company_email"`
	CompanyPhone            string  `json:"company_phone"`
	NRAStoreNumber          string  `json:"nra_store_number"`
	VATNumber               string  `json:"vat_number"`
	VATRate                 float64 `json:"vat_rate"`
}

type courierRequest struct {
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
	IsActive   bool   `json:"is_active"`
	SortOrder  int    `json:"sort_order"`
}

type courierResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
	IsActive   bool   `json:"is_active"`
	SortOrder  int    `json:"sort_order"`
}

// ── Converters ────────────────────────────────────────────────────────────────

func toListItem(inv domain.Invoice) invoiceListItem {
	item := invoiceListItem{
		ID:            inv.ID.String(),
		InvoiceNumber: inv.InvoiceNumber,
		DocumentType:  string(inv.DocumentType),
		OrderID:       inv.OrderID.String(),
		OrderNumber:   inv.OrderNumber,
		PaymentMethod: inv.PaymentMethod,
		RecipientName: inv.RecipientName,
		TotalInclVAT:  float64(inv.TotalInclVAT.AmountMinor) / 100,
		Currency:      inv.TotalInclVAT.Currency,
		CreatedAt:     inv.CreatedAt.UTC().Format(time.RFC3339),
	}
	if inv.StornoOfInvoiceID != nil {
		s := inv.StornoOfInvoiceID.String()
		item.StornoOfInvoiceID = &s
	}
	return item
}

func toDetailResponse(inv domain.Invoice) invoiceDetailResponse {
	resp := invoiceDetailResponse{
		invoiceListItem:       toListItem(inv),
		RecipientAddress:      inv.RecipientAddress,
		RecipientEmail:        inv.RecipientEmail,
		CompanyName:           inv.CompanyName,
		CompanyLegalType:      inv.CompanyLegalType,
		CompanyEIK:            inv.CompanyEIK,
		NRAStoreNumber:        inv.NRAStoreNumber,
		CardProvider:          inv.CardProvider,
		CardProviderReference: inv.CardProviderReference,
		CourierName:           inv.CourierName,
		CourierIdentifier:     inv.CourierIdentifier,
		SubtotalExclVAT:       float64(inv.SubtotalExclVAT.AmountMinor) / 100,
		VATAmount:             float64(inv.VATAmount.AmountMinor) / 100,
		VATRate:               inv.VATRate,
		DeliveryFee:           float64(inv.DeliveryFee.AmountMinor) / 100,
	}
	if inv.DiscountAmount != nil {
		v := float64(inv.DiscountAmount.AmountMinor) / 100
		resp.DiscountAmount = &v
	}
	for _, item := range inv.LineItems {
		resp.LineItems = append(resp.LineItems, lineItemResponse{
			ProductName:      item.ProductName,
			VariantLabel:     item.VariantLabel,
			NKSCode:          item.NKSCode,
			Quantity:         item.Quantity,
			UnitPriceInclVAT: float64(item.UnitPriceInclVAT.AmountMinor) / 100,
			UnitPriceExclVAT: float64(item.UnitPriceExclVAT.AmountMinor) / 100,
			VATPerUnit:       float64(item.VATPerUnit.AmountMinor) / 100,
			LineTotalInclVAT: float64(item.LineTotalInclVAT.AmountMinor) / 100,
			LineTotalExclVAT: float64(item.LineTotalExclVAT.AmountMinor) / 100,
			LineVATAmount:    float64(item.LineVATAmount.AmountMinor) / 100,
		})
	}
	return resp
}

func toSettingsResponse(s domain.InvoiceSettings) invoiceSettingsResponse {
	return invoiceSettingsResponse{
		CompanyName:             s.CompanyName,
		CompanyLegalType:        s.CompanyLegalType,
		CompanyEIK:              s.CompanyEIK,
		CompanyAddressStreet:     s.CompanyAddressStreet,
		CompanyAddressCity:       s.CompanyAddressCity,
		CompanyAddressPostalCode: s.CompanyAddressPostalCode,
		CompanyAddressCountry:    s.CompanyAddressCountry,
		CompanyEmail:            s.CompanyEmail,
		CompanyPhone:            s.CompanyPhone,
		NRAStoreNumber:          s.NRAStoreNumber,
		VATNumber:               s.VATNumber,
		VATRate:                 s.VATRate,
	}
}

func toCourierResponse(c domain.Courier) courierResponse {
	return courierResponse{
		ID:         c.ID.String(),
		Name:       c.Name,
		Identifier: c.Identifier,
		IsActive:   c.IsActive,
		SortOrder:  c.SortOrder,
	}
}

func actorFromContext(r *http.Request) string {
	if p, ok := authctx.FromContext(r.Context()); ok {
		return p.UserID.String()
	}
	return "system"
}

// ── HTML invoice renderer ─────────────────────────────────────────────────────

func renderInvoiceHTML(inv *domain.Invoice) ([]byte, error) {
	// Re-parse with all funcs to avoid template errors from init ordering
	tmpl := template.Must(template.New("invoice").Funcs(template.FuncMap{
		"formatMoney": func(minor int64) string {
			major := minor / 100
			cents := minor % 100
			if cents < 0 {
				cents = -cents
			}
			return fmt.Sprintf("%d.%02d", major, cents)
		},
		"sofiaTime": func(t time.Time) string {
			loc, _ := time.LoadLocation("Europe/Sofia")
			return t.In(loc).Format("02.01.2006 15:04")
		},
		"paymentLabel": func(method string) string {
			switch method {
			case "card_online":
				return "Онлайн плащане с карта"
			case "cash_on_delivery":
				return "Наложен платеж (пощенски превод)"
			case "card_on_easybox":
				return "Плащане с карта на EasyBox"
			default:
				return method
			}
		},
		"add1":      func(i int) int { return i + 1 },
		"delivExcl": func(minor int64) int64 { return minor * 100 / 120 },
		"delivVAT":  func(minor int64) int64 { return minor - minor*100/120 },
	}).Parse(invoiceHTMLTemplate))

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, inv); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

const invoiceHTMLTemplate = `<!DOCTYPE html>
<html lang="bg">
<head>
<meta charset="UTF-8">
<title>{{ if eq .DocumentType "сторно" }}СТОРНО {{ end }}ФАКТУРА № {{ .InvoiceNumber }}</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: Arial, sans-serif; font-size: 12px; color: #222; background: #fff; padding: 24px; }
  .no-print { margin-bottom: 16px; }
  @media print { .no-print { display: none; } body { padding: 0; } }
  .header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; border-bottom: 2px solid #000; padding-bottom: 16px; }
  .doc-title { font-size: 20px; font-weight: bold; text-align: right; }
  .doc-subtitle { font-size: 11px; color: #555; text-align: right; margin-top: 4px; }
  .parties { display: grid; grid-template-columns: 1fr 1fr; gap: 32px; margin-bottom: 20px; }
  .party h3 { font-size: 11px; text-transform: uppercase; color: #555; margin-bottom: 6px; }
  .party p { margin-bottom: 2px; }
  .meta { margin-bottom: 20px; font-size: 11px; color: #444; }
  .meta span { margin-right: 24px; }
  table { width: 100%; border-collapse: collapse; margin-bottom: 20px; font-size: 11px; }
  th { background: #f0f0f0; border: 1px solid #ccc; padding: 6px 8px; text-align: left; }
  td { border: 1px solid #ccc; padding: 5px 8px; }
  td.num { text-align: right; }
  .totals { margin-left: auto; width: 320px; font-size: 12px; }
  .totals table { margin-bottom: 0; }
  .totals td { border: none; padding: 3px 8px; }
  .totals .total-row { font-weight: bold; font-size: 14px; border-top: 1px solid #000; }
  .payment-footer { margin-top: 20px; padding-top: 12px; border-top: 1px solid #ccc; font-size: 11px; color: #444; }
  .storno-banner { background: #fee; border: 2px solid #c00; color: #c00; padding: 8px 16px; font-size: 13px; font-weight: bold; text-align: center; margin-bottom: 16px; }
</style>
</head>
<body>

<div class="no-print">
  <button onclick="window.print()">Принтирай / Запази като PDF</button>
</div>

{{ if eq .DocumentType "сторно" }}
<div class="storno-banner">СТОРНО ДОКУМЕНТ{{ if .StornoOfInvoiceID }} — анулира фактура{{ end }}</div>
{{ end }}

<div class="header">
  <div>
    <p><strong>{{ .CompanyName }} {{ .CompanyLegalType }}</strong></p>
    <p>ЕИК: {{ .CompanyEIK }}</p>
    {{ if .VATNumber }}<p>ДДС №: {{ .VATNumber }}</p>{{ end }}
    <p>{{ .CompanyAddress }}</p>
    <p>{{ .CompanyEmail }} | {{ .CompanyPhone }}</p>
    <p style="margin-top:6px;font-size:11px;color:#555">УНП: {{ .NRAStoreNumber }}</p>
  </div>
  <div>
    <div class="doc-title">{{ if eq .DocumentType "сторно" }}СТОРНО{{ else }}ФАКТУРА{{ end }}</div>
    <div class="doc-subtitle">№ {{ .InvoiceNumber }}</div>
    <div class="doc-subtitle">Дата: {{ sofiaTime .CreatedAt }}</div>
  </div>
</div>

<div class="parties">
  <div class="party">
    <h3>Продавач</h3>
    <p><strong>{{ .CompanyName }} {{ .CompanyLegalType }}</strong></p>
    <p>ЕИК: {{ .CompanyEIK }}</p>
    <p>{{ .CompanyAddress }}</p>
  </div>
  <div class="party">
    <h3>Купувач</h3>
    <p><strong>{{ .RecipientName }}</strong></p>
    <p>{{ .RecipientAddress }}</p>
    <p>{{ .RecipientEmail }}</p>
  </div>
</div>

<div class="meta">
  <span>Поръчка №: <strong>{{ .OrderNumber }}</strong></span>
  <span>Дата на поръчка: <strong>{{ sofiaTime .PlacedAt }}</strong></span>
  {{ if .CardProvider }}<span>Доставчик: <strong>{{ .CardProvider }}</strong></span>{{ end }}
  {{ if .CardProviderReference }}<span>Реф. транзакция: <strong>{{ .CardProviderReference }}</strong></span>{{ end }}
  {{ if .CourierName }}<span>Куриер: <strong>{{ .CourierName }}</strong></span>{{ end }}
</div>

<table>
  <thead>
    <tr>
      <th>#</th>
      <th>Наименование</th>
      <th>НКС код</th>
      <th>Кол.</th>
      <th>Ед. цена без ДДС</th>
      <th>ДДС/ед.</th>
      <th>Ред. сума с ДДС</th>
    </tr>
  </thead>
  <tbody>
    {{ range $i, $item := .LineItems }}
    <tr>
      <td>{{ add1 $i }}</td>
      <td>{{ $item.ProductName }}{{ if $item.VariantLabel }} — {{ $item.VariantLabel }}{{ end }}</td>
      <td>{{ $item.NKSCode }}</td>
      <td class="num">{{ $item.Quantity }}</td>
      <td class="num">{{ formatMoney $item.UnitPriceExclVAT.AmountMinor }}</td>
      <td class="num">{{ formatMoney $item.VATPerUnit.AmountMinor }}</td>
      <td class="num">{{ formatMoney $item.LineTotalInclVAT.AmountMinor }}</td>
    </tr>
    {{ end }}
    {{ if gt .DeliveryFee.AmountMinor 0 }}
    <tr>
      <td></td>
      <td>Доставка</td>
      <td></td>
      <td class="num">1</td>
      <td class="num">{{ formatMoney (delivExcl .DeliveryFee.AmountMinor) }}</td>
      <td class="num">{{ formatMoney (delivVAT .DeliveryFee.AmountMinor) }}</td>
      <td class="num">{{ formatMoney .DeliveryFee.AmountMinor }}</td>
    </tr>
    {{ end }}
  </tbody>
</table>

<div class="totals">
  <table>
    <tr><td>Данъчна основа:</td><td class="num">{{ formatMoney .SubtotalExclVAT.AmountMinor }} {{ .TotalInclVAT.Currency }}</td></tr>
    <tr><td>ДДС ({{ .VATRate }}%):</td><td class="num">{{ formatMoney .VATAmount.AmountMinor }} {{ .TotalInclVAT.Currency }}</td></tr>
    {{ if .DiscountAmount }}<tr><td>Отстъпка:</td><td class="num">-{{ formatMoney .DiscountAmount.AmountMinor }} {{ .TotalInclVAT.Currency }}</td></tr>{{ end }}
    <tr class="total-row"><td>ОБЩО ЗА ПЛАЩАНЕ:</td><td class="num">{{ formatMoney .TotalInclVAT.AmountMinor }} {{ .TotalInclVAT.Currency }}</td></tr>
  </table>
</div>

<div class="payment-footer">
  Начин на плащане: <strong>{{ paymentLabel .PaymentMethod }}</strong>
</div>

</body>
</html>`
