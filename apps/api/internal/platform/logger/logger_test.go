package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

// newTestLogger builds a JSON logger writing to buf using the same GCP handler
// wiring as New (which writes to stdout).
func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	inner := slog.NewJSONHandler(buf, &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		AddSource:   true,
		ReplaceAttr: gcpReplaceAttr,
	})
	return slog.New(&contextHandler{inner: inner})
}

func TestJSONHandlerEmitsGCPSchema(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	log.ErrorContext(context.Background(), "boom", slog.String("order_id", "abc"))

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("log line is not valid JSON: %v", err)
	}

	if rec["severity"] != "ERROR" {
		t.Errorf("severity = %v, want ERROR", rec["severity"])
	}
	if rec["message"] != "boom" {
		t.Errorf("message = %v, want boom", rec["message"])
	}
	if _, ok := rec["timestamp"]; !ok {
		t.Error("missing timestamp field")
	}
	if _, ok := rec["logging.googleapis.com/sourceLocation"]; !ok {
		t.Error("missing sourceLocation field")
	}
	if rec["order_id"] != "abc" {
		t.Errorf("order_id = %v, want abc", rec["order_id"])
	}
	// slog defaults must not leak through.
	if _, ok := rec["level"]; ok {
		t.Error("unexpected raw slog 'level' field")
	}
	if _, ok := rec["msg"]; ok {
		t.Error("unexpected raw slog 'msg' field")
	}
}

func TestContextAttrsAreAttached(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	ctx := WithAttrs(context.Background(), slog.String("request_id", "req-1"))
	ctx = WithAttrs(ctx, slog.String("logging.googleapis.com/trace", "projects/p/traces/t"))
	log.InfoContext(ctx, "request")

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("log line is not valid JSON: %v", err)
	}
	if rec["request_id"] != "req-1" {
		t.Errorf("request_id = %v, want req-1", rec["request_id"])
	}
	if rec["logging.googleapis.com/trace"] != "projects/p/traces/t" {
		t.Errorf("trace = %v, want projects/p/traces/t", rec["logging.googleapis.com/trace"])
	}
	if rec["severity"] != "INFO" {
		t.Errorf("severity = %v, want INFO", rec["severity"])
	}
}

func TestGCPSeverityMapping(t *testing.T) {
	cases := map[slog.Level]string{
		slog.LevelDebug:     "DEBUG",
		slog.LevelInfo:      "INFO",
		slog.LevelWarn:      "WARNING",
		slog.LevelError:     "ERROR",
		slog.LevelError + 4: "CRITICAL",
	}
	for lvl, want := range cases {
		if got := gcpSeverity(lvl); got != want {
			t.Errorf("gcpSeverity(%v) = %q, want %q", lvl, got, want)
		}
	}
}
