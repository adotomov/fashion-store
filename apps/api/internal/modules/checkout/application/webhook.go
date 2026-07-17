package application

import (
	"context"
	"time"
)

// Revolut webhook event types. We settle on ORDER_COMPLETED (auto-capture
// means completion is the money-moved signal) and release stock on the
// cancelled/failed events. ORDER_AUTHORISED is informational under auto-capture.
const (
	WebhookOrderCompleted     = "ORDER_COMPLETED"
	WebhookOrderAuthorised    = "ORDER_AUTHORISED"
	WebhookOrderCancelled     = "ORDER_CANCELLED"
	WebhookOrderPaymentFailed = "ORDER_PAYMENT_FAILED"
)

// Sweeper defaults: how often to reconcile, and how long an order may sit in
// pending_payment before it's treated as abandoned. Kept as constants (not
// env) — the values only matter for the safety net, not day-to-day flow.
const (
	DefaultPaymentSweepInterval = 5 * time.Minute
	DefaultAbandonedPaymentTTL  = 30 * time.Minute
)

// WebhookEvent is a verified, parsed Revolut webhook. ID is the idempotency
// key (a hash of the raw body), so a redelivery of the same event is deduped.
type WebhookEvent struct {
	ID              string
	Type            string
	ProviderOrderID string
	RawPayload      []byte
}

// WebhookEventStore is the idempotency ledger. Seen reports whether an event id
// has already been fully processed; Record persists it after success.
type WebhookEventStore interface {
	Seen(ctx context.Context, eventID string) (bool, error)
	Record(ctx context.Context, event WebhookEvent) error
}

// HandleWebhook processes one verified Revolut webhook. Idempotent: an event
// already recorded is skipped, and dispatch itself (FinalizePaidOrder /
// FailPayment) is safe to re-run. The event is recorded only after successful
// processing, so a transient failure returns an error (the caller replies 5xx)
// and Revolut's redelivery gets reprocessed rather than silently dropped.
func (s *Service) HandleWebhook(ctx context.Context, event WebhookEvent) error {
	if event.ProviderOrderID == "" {
		return nil // nothing to act on
	}
	seen, err := s.events.Seen(ctx, event.ID)
	if err != nil {
		s.logger.Error("webhook: dedup lookup failed", "error", err, "event", event.Type, "provider_order_id", event.ProviderOrderID)
		return err
	}
	if seen {
		return nil
	}

	switch event.Type {
	case WebhookOrderCompleted:
		if err := s.FinalizePaidOrder(ctx, event.ProviderOrderID); err != nil {
			s.logger.Error("webhook: finalize failed", "error", err, "event", event.Type, "provider_order_id", event.ProviderOrderID)
			return err
		}
	case WebhookOrderCancelled, WebhookOrderPaymentFailed:
		if err := s.FailPayment(ctx, event.ProviderOrderID, event.Type); err != nil {
			s.logger.Error("webhook: fail-payment failed", "error", err, "event", event.Type, "provider_order_id", event.ProviderOrderID)
			return err
		}
	default:
		// ORDER_AUTHORISED and any other event: no state change under
		// auto-capture. Still recorded below so we don't reprocess it.
	}

	if err := s.events.Record(ctx, event); err != nil {
		s.logger.Error("webhook: record failed", "error", err, "event", event.Type, "provider_order_id", event.ProviderOrderID)
		return err
	}
	return nil
}

// RunPaymentSweeper reconciles abandoned card payments on an interval until the
// context is cancelled — the safety net for missed/lost webhooks. Started as a
// goroutine alongside the API server.
func (s *Service) RunPaymentSweeper(ctx context.Context, interval, ttl time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.SweepAbandonedCardPayments(ctx, ttl); err != nil {
				s.logger.Error("payment sweep failed", "error", err)
			}
		}
	}
}

// SweepAbandonedCardPayments finds pending_payment orders older than ttl and
// asks Revolut for their authoritative state: a completed order that missed
// its webhook is finalized; anything else is failed and its stock released.
func (s *Service) SweepAbandonedCardPayments(ctx context.Context, ttl time.Duration) error {
	cutoff := time.Now().Add(-ttl)
	refs, err := s.orders.ListPendingPaymentOlderThan(ctx, cutoff)
	if err != nil {
		return err
	}
	for _, ref := range refs {
		paymentOrder, err := s.payments.GetOrder(ctx, ref.ProviderOrderID)
		if err != nil {
			s.logger.Error("sweep: failed to fetch revolut order", "error", err, "provider_order_id", ref.ProviderOrderID)
			continue
		}
		if paymentOrder.State == PaymentStateCompleted {
			if err := s.FinalizePaidOrder(ctx, ref.ProviderOrderID); err != nil {
				s.logger.Error("sweep: failed to finalize recovered order", "error", err, "order_id", ref.OrderID)
			}
			continue
		}
		if err := s.FailPayment(ctx, ref.ProviderOrderID, "abandoned"); err != nil {
			s.logger.Error("sweep: failed to fail abandoned order", "error", err, "order_id", ref.OrderID)
		}
	}
	return nil
}
