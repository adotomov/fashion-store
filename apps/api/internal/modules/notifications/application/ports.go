package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/notifications/domain"
)

// Repository persists the outbox. Claiming is deliberately lease-based rather
// than a plain "select pending": ClaimDue flips rows to 'sending' and pushes
// their next attempt forward, so a dispatcher that dies mid-send leaves rows
// that become due again on their own instead of stranding them.
type Repository interface {
	// Enqueue inserts a message, reporting false (without error) when the
	// dedupe key already exists — that is the normal idempotent no-op, not a
	// failure.
	Enqueue(ctx context.Context, msg domain.Message) (bool, error)
	// ClaimDue leases up to limit due messages for this dispatcher.
	ClaimDue(ctx context.Context, limit int, lease time.Duration) ([]domain.Message, error)
	MarkSent(ctx context.Context, id uuid.UUID, providerMessageID string) error
	// MarkRetry returns a message to 'pending' with a later next_attempt_at.
	MarkRetry(ctx context.Context, id uuid.UUID, nextAttemptAt time.Time, reason string) error
	// MarkFailed dead-letters a message whose retries are exhausted.
	MarkFailed(ctx context.Context, id uuid.UUID, reason string) error
	MarkSuppressed(ctx context.Context, id uuid.UUID, reason string) error
	// MarkDeliveryFailure settles a message the provider later told us it could
	// not deliver. Matched on the id the provider returned at send time.
	MarkDeliveryFailure(ctx context.Context, providerMessageID, reason string) error
}

// TemplateStore reads the per-locale copy for a template key.
type TemplateStore interface {
	Get(ctx context.Context, key, locale string) (domain.Template, error)
}

// Renderer turns a template plus the message payload into a sendable email,
// wrapping the fragment in the shared branded layout.
type Renderer interface {
	Render(ctx context.Context, tmpl domain.Template, vars map[string]any) (domain.Rendered, error)
}

// SendRequest is provider-agnostic; the From identity is owned by the sender
// implementation (it is fixed per environment), not chosen per message.
type SendRequest struct {
	ToEmail string
	ToName  string
	Subject string
	HTML    string
	Text    string
}

// EmailSender is the outbound port to an email provider. Implementations:
// SendGridSender (real) and LogSender (local/devbox, renders and logs only).
type EmailSender interface {
	Send(ctx context.Context, req SendRequest) (providerMessageID string, err error)
}

// SuppressionStore tracks addresses that must never be mailed again — hard
// bounces and spam complaints. IsSuppressed is consulted immediately before
// every send, so a suppression added after a message was queued still stops it.
type SuppressionStore interface {
	IsSuppressed(ctx context.Context, email string) (bool, error)
	Suppress(ctx context.Context, email, reason, detail string) error
}

// BrandingProvider supplies store-level context for the email layout. Backed by
// store settings so emails follow the storefront's name and logo.
type BrandingProvider interface {
	Branding(ctx context.Context) (domain.Branding, error)
}
