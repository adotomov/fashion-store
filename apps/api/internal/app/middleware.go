package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/adotomov/fashion-store/apps/api/internal/platform/logger"
	"go.opentelemetry.io/otel/trace"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// RequestID injects a unique request ID into the request context and
// response headers, and seeds it as a log attribute so every log line emitted
// during the request (via the *Context slog methods) carries request_id.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = newRequestID()
		}
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		ctx = logger.WithAttrs(ctx, slog.String("request_id", id))
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LogContext enriches the request context with the log attributes used for
// Cloud Logging correlation — method, path, and the GCP trace/span fields —
// so every log line for a request is grouped under its trace in Logs Explorer.
// The trace id is taken from the active OTel span when tracing is enabled, and
// otherwise from Cloud Run's X-Cloud-Trace-Context header. Must run after
// RequestID and (when enabled) the otelhttp span middleware.
func LogContext(projectID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
			}
			if traceID, spanID := traceContext(ctx, r); traceID != "" {
				if projectID != "" {
					attrs = append(attrs, slog.String(
						"logging.googleapis.com/trace",
						fmt.Sprintf("projects/%s/traces/%s", projectID, traceID),
					))
				}
				if spanID != "" {
					attrs = append(attrs, slog.String("logging.googleapis.com/spanId", spanID))
				}
			}
			next.ServeHTTP(w, r.WithContext(logger.WithAttrs(ctx, attrs...)))
		})
	}
}

// traceContext resolves the trace and span ids for log correlation, preferring
// the active OTel span (present once otelhttp has started one) and falling back
// to Cloud Run's X-Cloud-Trace-Context header ("TRACE_ID/SPAN_ID;o=1").
func traceContext(ctx context.Context, r *http.Request) (traceID, spanID string) {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		return sc.TraceID().String(), sc.SpanID().String()
	}
	header := r.Header.Get("X-Cloud-Trace-Context")
	if header == "" {
		return "", ""
	}
	if i := strings.IndexByte(header, ';'); i >= 0 {
		header = header[:i]
	}
	if i := strings.IndexByte(header, '/'); i >= 0 {
		return header[:i], header[i+1:]
	}
	return header, ""
}

// RequestIDFromContext returns the request ID stored on the context, if any.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

func newRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Recover catches panics in downstream handlers, logs them, and returns a
// 500 instead of crashing the process.
func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					// request_id and trace are attached to the context by the
					// RequestID/LogContext middleware, so ErrorContext emits them.
					log.ErrorContext(r.Context(), "panic recovered", slog.Any("error", rec))
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// RequestLogging logs structured information about each request.
func RequestLogging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			// request_id, trace, method, and path come from the context attrs
			// seeded by RequestID/LogContext, so this line only adds what is
			// specific to the completed response.
			log.InfoContext(r.Context(), "request",
				slog.Int("status", rec.status),
				slog.Duration("duration", time.Since(start)),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
			)
		})
	}
}

// CORS reflects the request Origin only when it is on the configured
// allowlist. It fails closed: an empty allowlist allows no cross-origin
// requests at all (rather than reflecting any origin), so a misconfigured or
// missing CORS_ALLOWED_ORIGINS can never silently open credentialed access.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if _, ok := allowed[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeaders sets response headers that harden the API against common
// web attacks. Because this is a JSON API that never serves scripts, is never
// framed, and never needs a referrer, the policy is maximally restrictive.
// HSTS is only emitted when enableHSTS is true (deployed HTTPS environments) —
// on plain-HTTP local dev it is pointless and browsers ignore it anyway.
func SecurityHeaders(enableHSTS bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			if enableHSTS {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

// MaxBodyBytes caps request body size to guard against memory-exhaustion via
// oversized payloads. The limit is set above the media-upload cap (10 MiB,
// enforced per-route in the catalog handlers) so uploads still work while
// abusive multi-hundred-megabyte requests are rejected with 413.
func MaxBodyBytes(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}
			next.ServeHTTP(w, r)
		})
	}
}
