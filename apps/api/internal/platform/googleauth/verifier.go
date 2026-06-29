package googleauth

import (
	"context"
	"fmt"

	"google.golang.org/api/idtoken"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/auth/application"
)

// Verifier verifies Google ID tokens against a configured OAuth client ID.
// It isolates the Google SDK from application/domain logic.
type Verifier struct {
	clientID string
}

func NewVerifier(clientID string) *Verifier {
	return &Verifier{clientID: clientID}
}

func (v *Verifier) Verify(ctx context.Context, idToken string) (application.GoogleIdentity, error) {
	payload, err := idtoken.Validate(ctx, idToken, v.clientID)
	if err != nil {
		return application.GoogleIdentity{}, fmt.Errorf("verify google id token: %w", err)
	}

	email, _ := payload.Claims["email"].(string)
	emailVerified, _ := payload.Claims["email_verified"].(bool)
	name, _ := payload.Claims["name"].(string)

	return application.GoogleIdentity{
		Subject:       payload.Subject,
		Email:         email,
		EmailVerified: emailVerified,
		FullName:      name,
	}, nil
}
