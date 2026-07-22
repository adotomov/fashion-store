package http

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"strings"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/notifications/application"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
)

const (
	sendGridSignatureHeader = "X-Twilio-Email-Event-Webhook-Signature"
	sendGridTimestampHeader = "X-Twilio-Email-Event-Webhook-Timestamp"
	// maxWebhookBody caps how much we read from an unauthenticated endpoint.
	// SendGrid batches events, so this is generous but still bounded.
	maxWebhookBody = 5 << 20 // 5 MiB
)

// sendGridEvent is one entry of the JSON array SendGrid POSTs.
type sendGridEvent struct {
	Event       string `json:"event"`
	Email       string `json:"email"`
	SGMessageID string `json:"sg_message_id"`
	Reason      string `json:"reason"`
	// Type distinguishes a permanent bounce ("bounced") from a transient block
	// ("blocked"); present on bounce events only.
	Type string `json:"type"`
}

// sendGridWebhook ingests delivery events. It is unauthenticated in the routing
// sense — the ECDSA signature over the raw body IS its authentication, so the
// signature check must happen before anything else touches the payload.
//
// Always replies 2xx once events are accepted; a 5xx makes SendGrid redeliver
// the whole batch, which we only want for genuinely transient failures.
func (h *Handler) sendGridWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBody))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "could not read request body")
		return
	}

	if !h.verifySignature(r, body) {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_signature", "signature verification failed")
		return
	}

	var events []sendGridEvent
	if err := json.Unmarshal(body, &events); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "request body is invalid")
		return
	}

	for _, ev := range events {
		normalised := application.ProviderEvent{
			Type:              ev.Event,
			Email:             ev.Email,
			ProviderMessageID: baseMessageID(ev.SGMessageID),
			Reason:            ev.Reason,
			BounceType:        ev.Type,
		}
		if err := h.service.HandleProviderEvent(r.Context(), normalised); err != nil {
			// Transient (database) failure — ask SendGrid to redeliver the batch.
			// Handling is idempotent, so reprocessing the earlier events is safe.
			httpx.WriteError(w, http.StatusInternalServerError, "processing_failed", "could not process events")
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// baseMessageID strips the per-recipient suffix SendGrid appends to the id it
// returned at send time ("<id>.filterdrecv-..."), so it matches what we stored.
func baseMessageID(sgMessageID string) string {
	if idx := strings.Index(sgMessageID, "."); idx >= 0 {
		return sgMessageID[:idx]
	}
	return sgMessageID
}

// verifySignature checks SendGrid's Signed Event Webhook: an ECDSA signature
// over timestamp+body, verified with the public key from the SendGrid console.
// Fails closed — no configured key, missing headers, or a malformed signature
// all reject the request.
func (h *Handler) verifySignature(r *http.Request, body []byte) bool {
	if h.verificationKey == nil {
		return false
	}
	signature := r.Header.Get(sendGridSignatureHeader)
	timestamp := r.Header.Get(sendGridTimestampHeader)
	if signature == "" || timestamp == "" {
		return false
	}

	sig, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}

	// SendGrid signs the timestamp concatenated with the raw request body.
	digest := sha256.Sum256(append([]byte(timestamp), body...))

	var parsed struct{ R, S *big.Int }
	if _, err := asn1.Unmarshal(sig, &parsed); err != nil {
		return false
	}
	return ecdsa.Verify(h.verificationKey, digest[:], parsed.R, parsed.S)
}
