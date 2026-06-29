package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/auth/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/auth/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/authctx"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
)

type Handler struct {
	service *application.Service
	logger  *slog.Logger
}

func NewHandler(service *application.Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/google", h.loginWithGoogle)
		r.Post("/refresh", h.refresh)
		r.Post("/logout", h.logout)
	})
}

type googleLoginRequest struct {
	IDToken string `json:"id_token"`
}

type sessionResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

func (h *Handler) loginWithGoogle(w http.ResponseWriter, r *http.Request) {
	var req googleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IDToken == "" {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "id_token is required")
		return
	}

	result, err := h.service.LoginWithGoogle(r.Context(), req.IDToken)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "google login failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_token", "could not verify Google ID token")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, sessionResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt.Format(httpTimeFormat),
	})
}

type refreshRequest struct {
	Token string `json:"token"`
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "token is required")
		return
	}

	result, err := h.service.Refresh(r.Context(), req.Token)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, sessionResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt.Format(httpTimeFormat),
	})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token == "" {
		var req refreshRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		token = req.Token
	}
	if token == "" {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "token is required")
		return
	}

	if err := h.service.Logout(r.Context(), token); err != nil {
		writeAuthError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

const httpTimeFormat = "2006-01-02T15:04:05Z07:00"

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if strings.HasPrefix(header, prefix) {
		return strings.TrimPrefix(header, prefix)
	}
	return ""
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrSessionNotFound), errors.Is(err, domain.ErrSessionExpired):
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_session", "session is invalid or expired")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}

// RequireAuth returns middleware that authenticates requests via Bearer
// token and injects the resulting principal into the request context.
func RequireAuth(service *application.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
				return
			}

			principal, err := service.Authenticate(r.Context(), token)
			if err != nil {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired session")
				return
			}

			ctx := authctx.WithPrincipal(r.Context(), principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth attempts Bearer authentication but never rejects the
// request — it injects a principal when the token is present and valid,
// and otherwise passes the request through unauthenticated. Used by routes
// that work for both signed-in and anonymous callers (e.g. the cart).
func OptionalAuth(service *application.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token != "" {
				if principal, err := service.Authenticate(r.Context(), token); err == nil {
					r = r.WithContext(authctx.WithPrincipal(r.Context(), principal))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole returns middleware that rejects requests whose authenticated
// principal lacks the given role. Must run after RequireAuth so the
// principal is already in the request context.
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := authctx.FromContext(r.Context())
			if !ok || !principal.HasRole(role) {
				httpx.WriteError(w, http.StatusForbidden, "forbidden", "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
