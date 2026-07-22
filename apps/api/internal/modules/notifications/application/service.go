package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/notifications/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/platform/metrics"
)

const (
	// DefaultDispatchInterval is how often the dispatcher looks for due mail.
	// Emails are expected within seconds, not milliseconds, so a short poll is
	// plenty and keeps load off the database.
	DefaultDispatchInterval = 15 * time.Second
	// DefaultBatchSize caps how many messages one tick claims, bounding both the
	// lease window and the burst rate against the provider.
	DefaultBatchSize = 20
	// DefaultLease is how long a claimed message stays invisible to other
	// dispatchers. Comfortably longer than a provider call, short enough that a
	// crashed dispatcher's work is retried promptly.
	DefaultLease = 2 * time.Minute
	// MaxAttempts before a message is dead-lettered as 'failed'.
	MaxAttempts = 6

	retryBaseDelay = time.Minute
	retryMaxDelay  = time.Hour
)

// Service owns the outbox: producers call Enqueue, and a background dispatcher
// (RunDispatcher) renders and sends whatever is due.
type Service struct {
	repo          Repository
	templates     TemplateStore
	renderer      Renderer
	sender        EmailSender
	suppressions  SuppressionStore
	branding      BrandingProvider
	defaultLocale string
	logger        *slog.Logger
}

func NewService(
	repo Repository,
	templates TemplateStore,
	renderer Renderer,
	sender EmailSender,
	suppressions SuppressionStore,
	branding BrandingProvider,
	defaultLocale string,
	logger *slog.Logger,
) *Service {
	return &Service{
		repo:          repo,
		templates:     templates,
		renderer:      renderer,
		sender:        sender,
		suppressions:  suppressions,
		branding:      branding,
		defaultLocale: defaultLocale,
		logger:        logger,
	}
}

// EnqueueInput is what a producer supplies. Vars are the template variables for
// this specific event; store branding is merged in at render time.
type EnqueueInput struct {
	TemplateKey string
	Locale      string
	ToEmail     string
	ToName      string
	DedupeKey   string
	Category    string
	Vars        map[string]any
}

// Enqueue queues one email. It is idempotent on DedupeKey: enqueueing the same
// logical event twice is a no-op, which is what makes it safe to call from a
// retried request or a redelivered webhook.
func (s *Service) Enqueue(ctx context.Context, input EnqueueInput) error {
	locale := input.Locale
	if locale == "" {
		locale = s.storeLocale(ctx)
	}
	vars := input.Vars
	if vars == nil {
		vars = map[string]any{}
	}

	msg := domain.Message{
		TemplateKey: input.TemplateKey,
		Locale:      locale,
		ToEmail:     input.ToEmail,
		ToName:      input.ToName,
		Payload:     vars,
		DedupeKey:   input.DedupeKey,
		Category:    input.Category,
	}
	if err := msg.Validate(); err != nil {
		return err
	}

	inserted, err := s.repo.Enqueue(ctx, msg)
	if err != nil {
		return err
	}
	if !inserted {
		s.logger.DebugContext(ctx, "email already queued for this event, skipping",
			slog.String("template_key", msg.TemplateKey),
			slog.String("dedupe_key", msg.DedupeKey))
		return nil
	}
	s.logger.InfoContext(ctx, "email queued",
		slog.String("template_key", msg.TemplateKey),
		slog.String("dedupe_key", msg.DedupeKey))
	return nil
}

// storeLocale resolves the language to mail a customer in when the producer has
// no better signal. Falls back to the guaranteed-seeded default locale if store
// settings can't be read — a message in the wrong language beats no message.
func (s *Service) storeLocale(ctx context.Context) string {
	if s.branding == nil {
		return s.defaultLocale
	}
	branding, err := s.branding.Branding(ctx)
	if err != nil || branding.Locale == "" {
		return s.defaultLocale
	}
	return branding.Locale
}

// RunDispatcher polls for due messages until ctx is cancelled. Started as a
// goroutine alongside the other background workers in cmd/api.
func (s *Service) RunDispatcher(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = DefaultDispatchInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.DispatchDue(ctx); err != nil {
				s.logger.ErrorContext(ctx, "email dispatch failed", slog.Any("error", err))
			}
		}
	}
}

// DispatchDue claims and processes one batch, returning how many it handled.
// Exported so a test (or an admin action) can drive a single pass.
func (s *Service) DispatchDue(ctx context.Context) (int, error) {
	messages, err := s.repo.ClaimDue(ctx, DefaultBatchSize, DefaultLease)
	if err != nil {
		return 0, fmt.Errorf("claim due emails: %w", err)
	}
	for _, msg := range messages {
		s.deliver(ctx, msg)
	}
	return len(messages), nil
}

// deliver runs one message to a terminal state. Failures are recorded on the
// row rather than returned, so one bad message never stalls the batch.
func (s *Service) deliver(ctx context.Context, msg domain.Message) {
	log := s.logger.With(
		slog.String("email_message_id", msg.ID.String()),
		slog.String("template_key", msg.TemplateKey),
	)

	// Checked per attempt (not at enqueue) so a suppression added after queueing
	// still stops the send.
	if s.suppressions != nil {
		suppressed, err := s.suppressions.IsSuppressed(ctx, msg.ToEmail)
		if err != nil {
			s.retryOrFail(ctx, msg, fmt.Errorf("suppression lookup: %w", err), log)
			return
		}
		if suppressed {
			if err := s.repo.MarkSuppressed(ctx, msg.ID, "recipient is on the suppression list"); err != nil {
				log.ErrorContext(ctx, "could not mark email suppressed", slog.Any("error", err))
			}
			metrics.EmailSuppressed(ctx)
			log.InfoContext(ctx, "email suppressed, not sent")
			return
		}
	}

	tmpl, err := s.lookupTemplate(ctx, msg.TemplateKey, msg.Locale)
	if err != nil {
		// Missing copy is a deploy/seed problem, not a transient one — retrying
		// six times would only delay the alert.
		s.failPermanently(ctx, msg, err, log)
		return
	}

	rendered, err := s.renderer.Render(ctx, tmpl, msg.Payload)
	if err != nil {
		// A template that cannot render will never render; dead-letter it.
		s.failPermanently(ctx, msg, fmt.Errorf("render: %w", err), log)
		return
	}

	providerID, err := s.sender.Send(ctx, SendRequest{
		ToEmail: msg.ToEmail,
		ToName:  msg.ToName,
		Subject: rendered.Subject,
		HTML:    rendered.HTML,
		Text:    rendered.Text,
	})
	if err != nil {
		s.retryOrFail(ctx, msg, err, log)
		return
	}

	if err := s.repo.MarkSent(ctx, msg.ID, providerID); err != nil {
		// The provider accepted it; losing the status write only risks a
		// duplicate on the next lease expiry, which we accept over losing mail.
		log.ErrorContext(ctx, "could not mark email sent", slog.Any("error", err))
		return
	}
	metrics.EmailSent(ctx, msg.TemplateKey)
	log.InfoContext(ctx, "email sent", slog.String("provider_message_id", providerID))
}

// lookupTemplate falls back to the default locale so a locale that has not been
// translated yet still sends (in English) rather than dead-lettering.
func (s *Service) lookupTemplate(ctx context.Context, key, locale string) (domain.Template, error) {
	tmpl, err := s.templates.Get(ctx, key, locale)
	if err == nil {
		return tmpl, nil
	}
	if !errors.Is(err, domain.ErrTemplateNotFound) || locale == s.defaultLocale {
		return domain.Template{}, err
	}
	return s.templates.Get(ctx, key, s.defaultLocale)
}

func (s *Service) retryOrFail(ctx context.Context, msg domain.Message, cause error, log *slog.Logger) {
	if msg.Attempts >= MaxAttempts {
		s.failPermanently(ctx, msg, fmt.Errorf("giving up after %d attempts: %w", msg.Attempts, cause), log)
		return
	}
	next := time.Now().Add(backoff(msg.Attempts))
	if err := s.repo.MarkRetry(ctx, msg.ID, next, cause.Error()); err != nil {
		log.ErrorContext(ctx, "could not schedule email retry", slog.Any("error", err))
		return
	}
	metrics.EmailFailed(ctx, "retry")
	log.WarnContext(ctx, "email send failed, will retry",
		slog.Int("attempts", msg.Attempts),
		slog.Time("next_attempt_at", next),
		slog.Any("error", cause))
}

func (s *Service) failPermanently(ctx context.Context, msg domain.Message, cause error, log *slog.Logger) {
	if err := s.repo.MarkFailed(ctx, msg.ID, cause.Error()); err != nil {
		log.ErrorContext(ctx, "could not mark email failed", slog.Any("error", err))
		return
	}
	metrics.EmailFailed(ctx, "dead_letter")
	log.ErrorContext(ctx, "email dead-lettered", slog.Any("error", cause))
}

// backoff grows exponentially from retryBaseDelay and is capped, so a provider
// outage backs off to hourly retries instead of hammering it.
func backoff(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	delay := time.Duration(float64(retryBaseDelay) * math.Pow(2, float64(attempts-1)))
	if delay > retryMaxDelay || delay <= 0 {
		return retryMaxDelay
	}
	return delay
}
