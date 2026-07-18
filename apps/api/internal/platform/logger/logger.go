package logger

import (
	"context"
	"log/slog"
	"os"
	"strconv"
)

// New builds a slog.Logger configured with the given level, format, service
// name, and environment. format is "json" or "text"; level is one of
// debug|info|warn|error.
//
// The JSON handler emits Cloud Logging's structured schema (severity, message,
// timestamp, sourceLocation) so Cloud Run's log ingestion colors by severity
// and filters correctly, and it is wrapped in a context handler that merges
// request-scoped attributes (request_id, trace, user_id) added via WithAttrs.
// The text handler (local dev) keeps slog's readable default layout.
func New(level, format, service, env string) *slog.Logger {
	lvl := parseLevel(level)

	var inner slog.Handler
	if format == "text" {
		inner = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     lvl,
			AddSource: true,
		})
	} else {
		inner = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:       lvl,
			AddSource:   true,
			ReplaceAttr: gcpReplaceAttr,
		})
	}

	return slog.New(&contextHandler{inner: inner}).With(
		slog.String("service", service),
		slog.String("env", env),
	)
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// gcpReplaceAttr remaps slog's default top-level keys to the field names Cloud
// Logging expects. Nested attrs (groups) are left untouched.
func gcpReplaceAttr(groups []string, a slog.Attr) slog.Attr {
	if len(groups) > 0 {
		return a
	}
	switch a.Key {
	case slog.LevelKey:
		a.Key = "severity"
		if lvl, ok := a.Value.Any().(slog.Level); ok {
			a.Value = slog.StringValue(gcpSeverity(lvl))
		}
	case slog.MessageKey:
		a.Key = "message"
	case slog.TimeKey:
		// slog's JSON handler already encodes time as RFC3339Nano, which Cloud
		// Logging parses from the "timestamp" field.
		a.Key = "timestamp"
	case slog.SourceKey:
		a.Key = "logging.googleapis.com/sourceLocation"
		if src, ok := a.Value.Any().(*slog.Source); ok {
			a.Value = slog.GroupValue(
				slog.String("file", src.File),
				slog.String("line", strconv.Itoa(src.Line)),
				slog.String("function", src.Function),
			)
		}
	}
	return a
}

// gcpSeverity maps slog levels onto Cloud Logging severity strings.
func gcpSeverity(l slog.Level) string {
	switch {
	case l < slog.LevelInfo:
		return "DEBUG"
	case l < slog.LevelWarn:
		return "INFO"
	case l < slog.LevelError:
		return "WARNING"
	case l < slog.LevelError+4:
		return "ERROR"
	default:
		return "CRITICAL"
	}
}

type ctxKey struct{}

// WithAttrs returns a context carrying the given attributes so that every log
// record emitted with this context (via the *Context slog methods) is
// automatically annotated with them. Attributes accumulate: calling WithAttrs
// again appends to any already present. Used by middleware to attach
// request_id / trace / user_id once per request instead of at every call site.
func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	if len(attrs) == 0 {
		return ctx
	}
	existing := attrsFromContext(ctx)
	merged := make([]slog.Attr, 0, len(existing)+len(attrs))
	merged = append(merged, existing...)
	merged = append(merged, attrs...)
	return context.WithValue(ctx, ctxKey{}, merged)
}

func attrsFromContext(ctx context.Context) []slog.Attr {
	if ctx == nil {
		return nil
	}
	if attrs, ok := ctx.Value(ctxKey{}).([]slog.Attr); ok {
		return attrs
	}
	return nil
}

// contextHandler decorates records with the request-scoped attributes stashed
// on the context by WithAttrs before delegating to the inner handler.
type contextHandler struct {
	inner slog.Handler
}

func (h *contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if attrs := attrsFromContext(ctx); len(attrs) > 0 {
		r.AddAttrs(attrs...)
	}
	return h.inner.Handle(ctx, r)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{inner: h.inner.WithGroup(name)}
}
