// Package domain holds the notifications module's core types: the outbox
// message that represents one queued email, the per-locale template that gives
// it copy, and the rendered result handed to an email provider.
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Message lifecycle. A message is enqueued 'pending', leased as 'sending' while
// a dispatcher attempt is in flight, and settles as 'sent', 'failed' (retries
// exhausted — a dead letter) or 'suppressed' (the address is on the suppression
// list, so we deliberately never attempted delivery).
const (
	StatusPending    = "pending"
	StatusSending    = "sending"
	StatusSent       = "sent"
	StatusFailed     = "failed"
	StatusSuppressed = "suppressed"
)

// Template keys for the transactional set. These are the contract between a
// producer (which enqueues) and the seeded rows in email_templates.
const (
	TemplateWelcome           = "welcome"
	TemplateOrderConfirmation = "order_confirmation"
	TemplateShippingUpdate    = "shipping_update"
	TemplatePaymentFailed     = "payment_failed"
)

var (
	ErrTemplateNotFound = errors.New("email template not found")
	ErrMessageNotFound  = errors.New("email message not found")
)

type ValidationError string

func (e ValidationError) Error() string { return string(e) }

// Message is one queued email in the outbox.
type Message struct {
	ID          uuid.UUID
	TemplateKey string
	Locale      string
	ToEmail     string
	ToName      string
	// Payload carries the template variables. It is merged with store branding
	// at render time, so producers only supply what is specific to the event.
	Payload map[string]any
	// DedupeKey makes enqueueing idempotent — a unique index rejects a second
	// message for the same logical event (e.g. order_confirmation:<order_id>).
	DedupeKey string
	// Category is a nullable classification used for metrics only. Transactional
	// mail leaves it empty; it is the seam a future marketing stream would use.
	Category          string
	Status            string
	Attempts          int
	NextAttemptAt     time.Time
	LastError         string
	ProviderMessageID string
	SentAt            *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Template is the per-locale copy for one email. HTMLBody/TextBody are template
// fragments holding only the inner content — the renderer wraps them in the
// shared branded layout.
type Template struct {
	Key       string
	Locale    string
	Subject   string
	HTMLBody  string
	TextBody  string
	UpdatedAt time.Time
}

// Branding is the store-level context every email is rendered with, sourced
// from store settings so emails stay in sync with the storefront.
type Branding struct {
	StoreName string
	// Locale is the store's configured language. Producers that have no better
	// signal (orders and accounts carry no locale of their own) enqueue with an
	// empty locale and get this, so a Bulgarian store mails in Bulgarian.
	Locale        string
	LogoURL       string
	StorefrontURL string
	SupportEmail  string
	// PostalAddress is rendered in the footer. Kept available because a sender's
	// physical address is expected on commercial mail; harmless on transactional.
	PostalAddress string
}

// Rendered is a fully materialised email, ready to hand to a provider.
type Rendered struct {
	Subject string
	HTML    string
	Text    string
}

// Validate rejects a message that could never be delivered, so a bad producer
// fails loudly at enqueue time rather than silently dead-lettering later.
func (m Message) Validate() error {
	if m.TemplateKey == "" {
		return ValidationError("template_key is required")
	}
	if m.ToEmail == "" {
		return ValidationError("to_email is required")
	}
	if m.DedupeKey == "" {
		return ValidationError("dedupe_key is required")
	}
	if m.Locale == "" {
		return ValidationError("locale is required")
	}
	return nil
}
