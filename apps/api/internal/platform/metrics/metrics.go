// Package metrics defines the application's business metrics, exported to Cloud
// Monitoring via the OTel meter provider configured in platform/telemetry.
//
// The instrument set is deliberately small and low-cardinality: HTTP RED
// metrics (request count / latency / errors) are left to Cloud Run's free
// built-in metrics, and every attribute here is drawn from a bounded enum
// (payment method, failure reason, sweeper kind, webhook type/result) — never a
// per-user, per-order or otherwise unbounded value, which would multiply the
// billable time-series count.
//
// Instruments are created from the global meter provider at package init. The
// OTel global provider delegates to the real provider once telemetry.Setup
// installs it, so ordering is not a concern; when metrics are disabled the
// instruments are no-ops and recording is free.
package metrics

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

const scopeName = "github.com/adotomov/fashion-store/apps/api"

var (
	ordersPlaced         metric.Int64Counter
	paymentsInitiated    metric.Int64Counter
	paymentsSucceeded    metric.Int64Counter
	paymentsFailed       metric.Int64Counter
	refunds              metric.Int64Counter
	reservationConflicts metric.Int64Counter
	sweeperReclaims      metric.Int64Counter
	webhookEvents        metric.Int64Counter
	fulfillmentErrors    metric.Int64Counter
)

func init() {
	m := otel.Meter(scopeName)
	ordersPlaced = mustCounter(m, "orders_placed_total", "Orders successfully placed.")
	paymentsInitiated = mustCounter(m, "payments_initiated_total", "Card payments initiated (Revolut order created).")
	paymentsSucceeded = mustCounter(m, "payments_succeeded_total", "Card payments settled as paid.")
	paymentsFailed = mustCounter(m, "payments_failed_total", "Card payments that failed, cancelled or lapsed.")
	refunds = mustCounter(m, "refunds_total", "Refund attempts by outcome.")
	reservationConflicts = mustCounter(m, "checkout_reservation_conflicts_total", "Checkout session reservations rejected for insufficient stock.")
	sweeperReclaims = mustCounter(m, "checkout_sweeper_reclaims_total", "Stock reservations reclaimed by the background sweeper.")
	webhookEvents = mustCounter(m, "webhook_events_total", "Inbound payment webhooks by type and result.")
	fulfillmentErrors = mustCounter(m, "fulfillment_poll_errors_total", "Errors during the fulfillment tracking poll.")
}

// mustCounter creates a counter, falling back to a no-op instrument on the
// (practically impossible) creation error so callers never need a nil check.
func mustCounter(m metric.Meter, name, desc string) metric.Int64Counter {
	c, err := m.Int64Counter(name, metric.WithDescription(desc), metric.WithUnit("1"))
	if err != nil {
		// Unreachable in practice; keep callers nil-safe with a no-op counter.
		c, _ = noop.NewMeterProvider().Meter(scopeName).Int64Counter(name)
	}
	return c
}

// OrderPlaced records a placed order. paymentMethod is a bounded enum
// (cash_on_delivery|card_on_easybox|card_online).
func OrderPlaced(ctx context.Context, paymentMethod string) {
	ordersPlaced.Add(ctx, 1, metric.WithAttributes(attribute.String("payment_method", paymentMethod)))
}

// PaymentInitiated records a card payment being initiated.
func PaymentInitiated(ctx context.Context) {
	paymentsInitiated.Add(ctx, 1)
}

// PaymentSucceeded records a card payment settling as paid.
func PaymentSucceeded(ctx context.Context) {
	paymentsSucceeded.Add(ctx, 1)
}

// PaymentFailed records a card payment failure. reason is a bounded enum
// (declined|cancelled|expired|amount_mismatch|...).
func PaymentFailed(ctx context.Context, reason string) {
	paymentsFailed.Add(ctx, 1, metric.WithAttributes(attribute.String("reason", reason)))
}

// RefundRecorded records a refund attempt. status is a bounded enum
// (succeeded|failed).
func RefundRecorded(ctx context.Context, status string) {
	refunds.Add(ctx, 1, metric.WithAttributes(attribute.String("status", status)))
}

// ReservationConflict records a checkout session reservation rejected because
// the requested stock was unavailable.
func ReservationConflict(ctx context.Context) {
	reservationConflicts.Add(ctx, 1)
}

// SweeperReclaim records a stock reservation reclaimed by the background
// sweeper. kind is a bounded enum (session|abandoned_payment).
func SweeperReclaim(ctx context.Context, kind string) {
	sweeperReclaims.Add(ctx, 1, metric.WithAttributes(attribute.String("kind", kind)))
}

// WebhookEvent records an inbound payment webhook. eventType and result are
// bounded enums (result: verified|rejected|ignored).
func WebhookEvent(ctx context.Context, eventType, result string) {
	webhookEvents.Add(ctx, 1, metric.WithAttributes(
		attribute.String("type", eventType),
		attribute.String("result", result),
	))
}

// FulfillmentPollError records an error during the fulfillment tracking poll.
func FulfillmentPollError(ctx context.Context) {
	fulfillmentErrors.Add(ctx, 1)
}
