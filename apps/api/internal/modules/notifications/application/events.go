package application

import (
	"context"
	"log/slog"

	"github.com/adotomov/fashion-store/apps/api/internal/platform/metrics"
)

// Provider event types we act on. Anything else (processed, deferred, open,
// click) is recorded in the log and otherwise ignored.
const (
	EventDelivered  = "delivered"
	EventBounce     = "bounce"
	EventDropped    = "dropped"
	EventSpamReport = "spamreport"
)

// Bounce classifications. Only a permanent failure suppresses the address — a
// soft/blocked bounce is often transient (a full mailbox, greylisting) and
// suppressing on it would silently cut off a legitimate customer.
const (
	BounceTypePermanent = "bounced"
	BounceTypeBlocked   = "blocked"
)

// SuppressionReason values stored against a suppressed address.
const (
	SuppressionReasonBounce    = "bounce"
	SuppressionReasonComplaint = "complaint"
)

// ProviderEvent is one normalised delivery event from the email provider.
type ProviderEvent struct {
	Type              string
	Email             string
	ProviderMessageID string
	Reason            string
	// BounceType distinguishes a permanent bounce from a transient block.
	BounceType string
}

// HandleProviderEvent applies one delivery event: permanent failures suppress
// the address so we stop mailing it, and the originating message is settled as
// failed. Unknown event types are ignored rather than erroring, so a provider
// adding new events never causes webhook retries.
func (s *Service) HandleProviderEvent(ctx context.Context, ev ProviderEvent) error {
	log := s.logger.With(
		slog.String("provider_event", ev.Type),
		slog.String("provider_message_id", ev.ProviderMessageID),
	)

	switch ev.Type {
	case EventDelivered:
		log.DebugContext(ctx, "email delivered")
		return nil

	case EventSpamReport:
		// The recipient explicitly marked us as spam — the strongest possible
		// signal to stop, and left unhandled the single fastest way to ruin
		// deliverability for every other customer.
		if err := s.suppress(ctx, ev.Email, SuppressionReasonComplaint, ev.Reason); err != nil {
			return err
		}
		metrics.EmailFailed(ctx, "complaint")
		log.InfoContext(ctx, "recipient reported spam, address suppressed")
		return nil

	case EventBounce, EventDropped:
		// A blocked/soft bounce is usually transient, so it settles the message
		// without condemning the address.
		if ev.BounceType == BounceTypeBlocked {
			log.InfoContext(ctx, "soft bounce, address not suppressed")
			return s.recordDeliveryFailure(ctx, ev)
		}
		if err := s.suppress(ctx, ev.Email, SuppressionReasonBounce, ev.Reason); err != nil {
			return err
		}
		metrics.EmailFailed(ctx, "bounce")
		log.InfoContext(ctx, "hard bounce, address suppressed")
		return s.recordDeliveryFailure(ctx, ev)

	default:
		log.DebugContext(ctx, "ignoring provider event")
		return nil
	}
}

func (s *Service) suppress(ctx context.Context, email, reason, detail string) error {
	if s.suppressions == nil || email == "" {
		return nil
	}
	return s.suppressions.Suppress(ctx, email, reason, detail)
}

// recordDeliveryFailure marks the originating outbox row failed. A missing row
// is not an error: the provider may report on a message older than our
// retention, and failing here would only trigger pointless webhook retries.
func (s *Service) recordDeliveryFailure(ctx context.Context, ev ProviderEvent) error {
	if ev.ProviderMessageID == "" {
		return nil
	}
	reason := ev.Type
	if ev.Reason != "" {
		reason = ev.Type + ": " + ev.Reason
	}
	return s.repo.MarkDeliveryFailure(ctx, ev.ProviderMessageID, reason)
}
