package infrastructure

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/notifications/application"
)

// LogSender renders and logs an email instead of delivering it — the default
// locally and in devbox, so development needs no SendGrid account and no real
// mail can escape. Mirrors the fake Speedy client and the mock Revolut gateway.
//
// The full rendered body is logged at debug so a developer can inspect exact
// markup, while info stays a one-line summary.
type LogSender struct {
	logger *slog.Logger
}

func NewLogSender(logger *slog.Logger) *LogSender {
	return &LogSender{logger: logger}
}

func (s *LogSender) Send(ctx context.Context, req application.SendRequest) (string, error) {
	id := "log_" + uuid.NewString()
	s.logger.InfoContext(ctx, "email (not sent — log sender)",
		slog.String("to", req.ToEmail),
		slog.String("subject", req.Subject),
		slog.String("provider_message_id", id))
	s.logger.DebugContext(ctx, "email body",
		slog.String("to", req.ToEmail),
		slog.String("text", req.Text),
		slog.String("html", req.HTML))
	return id, nil
}
