package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/notifications/application"
)

const (
	sendGridEndpoint = "https://api.sendgrid.com/v3/mail/send"
	sendGridTimeout  = 20 * time.Second
)

// SendGridSender delivers mail through SendGrid's v3 API. The From identity is
// fixed per environment (info@verani.bg) rather than chosen per message, so no
// producer can accidentally send under a different, unauthenticated address.
type SendGridSender struct {
	apiKey    string
	fromEmail string
	fromName  string
	client    *http.Client
}

func NewSendGridSender(apiKey, fromEmail, fromName string, client *http.Client) *SendGridSender {
	if client == nil {
		client = &http.Client{Timeout: sendGridTimeout}
	}
	return &SendGridSender{apiKey: apiKey, fromEmail: fromEmail, fromName: fromName, client: client}
}

type sendGridAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type sendGridContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type sendGridPayload struct {
	Personalizations []struct {
		To []sendGridAddress `json:"to"`
	} `json:"personalizations"`
	From    sendGridAddress   `json:"from"`
	ReplyTo *sendGridAddress  `json:"reply_to,omitempty"`
	Subject string            `json:"subject"`
	Content []sendGridContent `json:"content"`
}

func (s *SendGridSender) Send(ctx context.Context, req application.SendRequest) (string, error) {
	payload := sendGridPayload{
		From:    sendGridAddress{Email: s.fromEmail, Name: s.fromName},
		Subject: req.Subject,
	}
	payload.Personalizations = append(payload.Personalizations, struct {
		To []sendGridAddress `json:"to"`
	}{To: []sendGridAddress{{Email: req.ToEmail, Name: req.ToName}}})

	// SendGrid requires text/plain first when both parts are present.
	if req.Text != "" {
		payload.Content = append(payload.Content, sendGridContent{Type: "text/plain", Value: req.Text})
	}
	payload.Content = append(payload.Content, sendGridContent{Type: "text/html", Value: req.HTML})

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal sendgrid payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, sendGridEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("call sendgrid: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Cap the body: a provider error page should not end up wholesale in the
		// outbox row or the logs.
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("sendgrid returned %d: %s", resp.StatusCode, bytes.TrimSpace(detail))
	}

	// SendGrid returns 202 with an empty body; X-Message-Id correlates this send
	// with the event webhook later.
	return resp.Header.Get("X-Message-Id"), nil
}
