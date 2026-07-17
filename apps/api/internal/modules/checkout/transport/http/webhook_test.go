package http

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func sign(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("v1." + timestamp + "." + string(body)))
	return "v1=" + hex.EncodeToString(mac.Sum(nil))
}

func newSignedRequest(sigHeader, timestamp string) *http.Request {
	r, _ := http.NewRequest(http.MethodPost, "/webhooks/revolut", nil)
	if sigHeader != "" {
		r.Header.Set(revolutSignatureHeader, sigHeader)
	}
	if timestamp != "" {
		r.Header.Set(revolutTimestampHeader, timestamp)
	}
	return r
}

func TestVerifyWebhookSignature(t *testing.T) {
	const secret = "wsk_test_secret"
	body := []byte(`{"event":"ORDER_COMPLETED","order_id":"abc123"}`)
	now := strconv.FormatInt(time.Now().UnixMilli(), 10)

	t.Run("valid signature passes", func(t *testing.T) {
		h := &Handler{webhookSecret: secret}
		r := newSignedRequest(sign(secret, now, body), now)
		if !h.verifyWebhookSignature(r, body) {
			t.Fatal("expected valid signature to verify")
		}
	})

	t.Run("valid among multiple space-separated signatures passes", func(t *testing.T) {
		h := &Handler{webhookSecret: secret}
		r := newSignedRequest("v1=deadbeef "+sign(secret, now, body), now)
		if !h.verifyWebhookSignature(r, body) {
			t.Fatal("expected verification to accept a rotated signature list")
		}
	})

	t.Run("wrong secret fails", func(t *testing.T) {
		h := &Handler{webhookSecret: secret}
		r := newSignedRequest(sign("other_secret", now, body), now)
		if h.verifyWebhookSignature(r, body) {
			t.Fatal("expected signature under a different secret to fail")
		}
	})

	t.Run("tampered body fails", func(t *testing.T) {
		h := &Handler{webhookSecret: secret}
		r := newSignedRequest(sign(secret, now, body), now)
		if h.verifyWebhookSignature(r, []byte(`{"event":"ORDER_COMPLETED","order_id":"tampered"}`)) {
			t.Fatal("expected a body that doesn't match the signature to fail")
		}
	})

	t.Run("stale timestamp fails", func(t *testing.T) {
		h := &Handler{webhookSecret: secret}
		old := strconv.FormatInt(time.Now().Add(-10*time.Minute).UnixMilli(), 10)
		r := newSignedRequest(sign(secret, old, body), old)
		if h.verifyWebhookSignature(r, body) {
			t.Fatal("expected a stale timestamp to be rejected")
		}
	})

	t.Run("empty configured secret fails closed", func(t *testing.T) {
		h := &Handler{webhookSecret: ""}
		r := newSignedRequest(sign(secret, now, body), now)
		if h.verifyWebhookSignature(r, body) {
			t.Fatal("expected verification to fail closed with no configured secret")
		}
	})

	t.Run("missing headers fail", func(t *testing.T) {
		h := &Handler{webhookSecret: secret}
		if h.verifyWebhookSignature(newSignedRequest("", ""), body) {
			t.Fatal("expected missing signature/timestamp headers to fail")
		}
	})
}
