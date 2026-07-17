package http

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/checkout/application"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
)

const (
	revolutSignatureHeader = "Revolut-Signature"
	revolutTimestampHeader = "Revolut-Request-Timestamp"
	// webhookMaxSkew rejects timestamps too far from now to blunt replay of a
	// captured request. Revolut sends the timestamp in unix milliseconds.
	webhookMaxSkew = 5 * time.Minute
)

// revolutWebhook verifies the signature over the raw body, dedupes, and hands
// the event to the checkout service. It replies 2xx once the event is handled
// (or safely ignored) and 5xx on a transient failure so Revolut retries.
func (h *Handler) revolutWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "could not read request body")
		return
	}

	if !h.verifyWebhookSignature(r, body) {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_signature", "signature verification failed")
		return
	}

	var payload struct {
		Event   string `json:"event"`
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	event := application.WebhookEvent{
		ID:              hashHex(body),
		Type:            payload.Event,
		ProviderOrderID: payload.OrderID,
		RawPayload:      body,
	}
	if err := h.service.HandleWebhook(r.Context(), event); err != nil {
		// Transient failure — let Revolut redeliver.
		httpx.WriteError(w, http.StatusInternalServerError, "processing_failed", "could not process webhook")
		return
	}
	w.WriteHeader(http.StatusOK)
}

// verifyWebhookSignature checks the Revolut-Signature header against an
// HMAC-SHA256 of "v1.{timestamp}.{rawBody}" keyed by the webhook signing
// secret, and rejects stale timestamps. Fails closed: no configured secret,
// missing headers, or a bad/old timestamp all return false.
func (h *Handler) verifyWebhookSignature(r *http.Request, body []byte) bool {
	if h.webhookSecret == "" {
		return false
	}
	sigHeader := r.Header.Get(revolutSignatureHeader)
	timestamp := r.Header.Get(revolutTimestampHeader)
	if sigHeader == "" || timestamp == "" {
		return false
	}
	if millis, err := strconv.ParseInt(timestamp, 10, 64); err == nil {
		if age := time.Since(time.UnixMilli(millis)); age > webhookMaxSkew || age < -webhookMaxSkew {
			return false
		}
	} else {
		return false
	}

	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write([]byte("v1." + timestamp + "." + string(body)))
	expected := "v1=" + hex.EncodeToString(mac.Sum(nil))

	// The header may carry multiple space-separated signatures (rotation).
	for _, candidate := range strings.Fields(sigHeader) {
		if hmac.Equal([]byte(candidate), []byte(expected)) {
			return true
		}
	}
	return false
}

func hashHex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
