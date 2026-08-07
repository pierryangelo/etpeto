package api

import (
	"context"
	"errors"
	"net/http"

	"time"

	"github.com/danielgtaylor/huma/v2"
	authdomain "github.com/etpeto/etpeto/internal/auth"
	"github.com/etpeto/etpeto/internal/platform/ratelimit"
)

type AuthService interface {
	Login(context.Context, string, string) (authdomain.LoginResult, error)
	Authenticate(context.Context, string) (authdomain.AuthenticatedUser, error)
	Logout(context.Context, string) error
}

type LoginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginInput struct {
	Body LoginBody
}

type SessionBody struct {
	User authdomain.AuthenticatedUser `json:"user"`
}

type LoginOutput struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
	Body      SessionBody
}

type SessionInput struct {
	Session string `cookie:"__Host-etpeto_session"`
}

type MutationInput struct {
	Session string `cookie:"__Host-etpeto_session"`
	CSRF    string `header:"X-Etpeto-Csrf"`
}

type SessionOutput struct {
	Body SessionBody
}

type LogoutOutput struct {
	Status    int
	SetCookie []http.Cookie `header:"Set-Cookie"`
}

// failedSignInRule holds back brute force against one account. It counts only
// failures, so someone who mistypes and then succeeds is not locked out, and it
// is keyed by address rather than by network: an attacker gains nothing by
// spreading attempts across addresses against the same account.
var failedSignInRule = ratelimit.Rule{Limit: 8, Window: 15 * time.Minute}

func registerAuth(api huma.API, authentication AuthService) {
	protected := guard{authentication: authentication}
	failedSignIns := ratelimit.New(failedSignInRule)
	huma.Register(api, huma.Operation{
		OperationID:  "local-login",
		Method:       http.MethodPost,
		Path:         "/api/v1/auth/local/login",
		Summary:      "Create an Etpeto session with local credentials",
		Tags:         []string{"Authentication"},
		MaxBodyBytes: 4 << 10,
	}, func(ctx context.Context, input *LoginInput) (*LoginOutput, error) {
		if authentication == nil {
			return nil, huma.Error503ServiceUnavailable("Autenticação temporariamente indisponível")
		}
		account := authdomain.NormalizeEmail(input.Body.Email)
		if blocked, _ := failedSignIns.Blocked(account); blocked {
			// The message is the same one an unknown account gets, so probing
			// cannot tell a locked account from a nonexistent one.
			return nil, huma.Error401Unauthorized("E-mail ou senha inválidos")
		}
		result, err := authentication.Login(ctx, input.Body.Email, input.Body.Password)
		if errors.Is(err, authdomain.ErrInvalidCredentials) {
			failedSignIns.Allow(account)
			return nil, huma.Error401Unauthorized("E-mail ou senha inválidos")
		}
		if err != nil {
			return nil, huma.Error503ServiceUnavailable("Autenticação temporariamente indisponível")
		}
		return &LoginOutput{
			SetCookie: []http.Cookie{
				activeSessionCookie(result.Token, result.ExpiresAt),
				activeCSRFCookie(result.CSRFToken, result.ExpiresAt),
			},
			Body: SessionBody{User: result.User},
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-auth-session",
		Method:      http.MethodGet,
		Path:        "/api/v1/auth/session",
		Summary:     "Describe the current Etpeto session",
		Tags:        []string{"Authentication"},
	}, func(ctx context.Context, input *SessionInput) (*SessionOutput, error) {
		user, err := protected.authenticate(ctx, input.Session)
		if err != nil {
			return nil, err
		}
		return &SessionOutput{Body: SessionBody{User: user}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "logout",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/logout",
		Summary:     "Revoke the current Etpeto session",
		Tags:        []string{"Authentication"},
	}, func(ctx context.Context, input *MutationInput) (*LogoutOutput, error) {
		cleared := &LogoutOutput{
			Status:    http.StatusNoContent,
			SetCookie: []http.Cookie{expiredSessionCookie(), expiredCSRFCookie()},
		}
		// Signing out an already invalid session is not a state change, so it
		// stays idempotent and still clears the browser cookies.
		if _, err := protected.authenticate(ctx, input.Session); err != nil {
			var status huma.StatusError
			if errors.As(err, &status) && status.GetStatus() == http.StatusUnauthorized {
				return cleared, nil
			}
			return nil, err
		}
		if _, err := protected.authorizeMutation(ctx, input.Session, input.CSRF); err != nil {
			return nil, err
		}
		if err := authentication.Logout(ctx, input.Session); err != nil {
			return nil, huma.Error503ServiceUnavailable("Autenticação temporariamente indisponível")
		}
		return cleared, nil
	})
}
