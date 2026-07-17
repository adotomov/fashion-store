package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/checkout/application"
)

const (
	defaultRevolutAPIVersion = "2024-09-01"
	revolutRequestTimeout    = 15 * time.Second
	revolutMaxResponseBytes  = 1 << 20
)

// RevolutGateway is the live Merchant API client. It authenticates with the
// secret API key as a Bearer token and pins the API version via header. Orders
// are created with automatic capture and the store's own order number as the
// merchant external reference, so a webhook (or a manual reconciliation) can
// always tie a Revolut order back to ours.
type RevolutGateway struct {
	baseURL    string
	apiKey     string
	apiVersion string
	http       *http.Client
	logger     *slog.Logger
}

func NewRevolutGateway(baseURL, apiKey, apiVersion string, logger *slog.Logger) *RevolutGateway {
	if apiVersion == "" {
		apiVersion = defaultRevolutAPIVersion
	}
	return &RevolutGateway{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		apiVersion: apiVersion,
		http:       &http.Client{Timeout: revolutRequestTimeout},
		logger:     logger,
	}
}

type revolutOrder struct {
	ID       string `json:"id"`
	Token    string `json:"token"`
	State    string `json:"state"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

func (o revolutOrder) toPaymentOrder() application.PaymentOrder {
	return application.PaymentOrder{
		ID:          o.ID,
		Token:       o.Token,
		State:       strings.ToLower(o.State),
		AmountMinor: o.Amount,
		Currency:    o.Currency,
	}
}

func (g *RevolutGateway) CreateOrder(ctx context.Context, input application.CreatePaymentOrderInput) (application.PaymentOrder, error) {
	body := map[string]any{
		"amount":                 input.Amount.AmountMinor,
		"currency":               input.Amount.Currency,
		"capture_mode":           "automatic",
		"merchant_order_ext_ref": input.OrderNumber,
	}
	if input.CustomerEmail != "" {
		body["customer"] = map[string]any{"email": input.CustomerEmail}
	}
	var resp revolutOrder
	if err := g.do(ctx, http.MethodPost, "/api/orders", body, &resp); err != nil {
		return application.PaymentOrder{}, err
	}
	return resp.toPaymentOrder(), nil
}

func (g *RevolutGateway) GetOrder(ctx context.Context, providerOrderID string) (application.PaymentOrder, error) {
	var resp revolutOrder
	if err := g.do(ctx, http.MethodGet, "/api/orders/"+providerOrderID, nil, &resp); err != nil {
		return application.PaymentOrder{}, err
	}
	return resp.toPaymentOrder(), nil
}

type revolutRefund struct {
	ID    string `json:"id"`
	State string `json:"state"`
}

func (g *RevolutGateway) Refund(ctx context.Context, input application.RefundInput) (application.RefundResult, error) {
	body := map[string]any{
		"amount":   input.Amount.AmountMinor,
		"currency": input.Amount.Currency,
	}
	if input.Reason != "" {
		body["description"] = input.Reason
	}
	var resp revolutRefund
	if err := g.do(ctx, http.MethodPost, "/api/orders/"+input.ProviderOrderID+"/refund", body, &resp); err != nil {
		return application.RefundResult{}, err
	}
	return application.RefundResult{ID: resp.ID, State: strings.ToLower(resp.State)}, nil
}

func (g *RevolutGateway) do(ctx context.Context, method, path string, reqBody, out any) error {
	var reader io.Reader
	if reqBody != nil {
		encoded, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, g.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	req.Header.Set("Revolut-Api-Version", g.apiVersion)
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := g.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, revolutMaxResponseBytes))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		g.logger.Error("revolut api error", "method", method, "path", path, "status", resp.StatusCode, "body", string(data))
		return fmt.Errorf("revolut api %s %s: status %d", method, path, resp.StatusCode)
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return err
		}
	}
	return nil
}
