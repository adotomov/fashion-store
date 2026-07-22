package http

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"errors"

	"github.com/go-chi/chi/v5"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/notifications/application"
)

type Handler struct {
	service *application.Service
	// verificationKey is nil when unconfigured, which makes the webhook reject
	// everything rather than trusting unsigned input.
	verificationKey *ecdsa.PublicKey
}

// NewHandler builds the webhook handler. The verification key is SendGrid's
// base64 DER public key; an empty or unparseable value leaves the endpoint
// failing closed, with the parse error returned so startup can log it.
func NewHandler(service *application.Service, verificationKey string) (*Handler, error) {
	h := &Handler{service: service}
	if verificationKey == "" {
		return h, nil
	}
	key, err := parseECDSAPublicKey(verificationKey)
	if err != nil {
		return h, err
	}
	h.verificationKey = key
	return h, nil
}

func parseECDSAPublicKey(encoded string) (*ecdsa.PublicKey, error) {
	der, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	parsed, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, err
	}
	key, ok := parsed.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("email webhook verification key is not an ECDSA public key")
	}
	return key, nil
}

// RegisterRoutes satisfies the RouteRegistrar interface. Notifications exposes
// nothing under /api/v1 — everything it serves is the signed webhook below.
func (h *Handler) RegisterRoutes(chi.Router) {}

// RegisterRootRoutes mounts the provider event webhook outside the /api/v1 tree
// and outside any auth middleware — the request signature is its authentication,
// exactly as the Revolut payment webhook is mounted.
func (h *Handler) RegisterRootRoutes(r chi.Router) {
	r.Post("/webhooks/sendgrid", h.sendGridWebhook)
}
