package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	authdomain "github.com/etpeto/etpeto/internal/auth"
)

const sessionCookieName = "__Host-etpeto_session"

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
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      SessionBody
}

type SessionInput struct {
	Session string `cookie:"__Host-etpeto_session"`
}

type SessionOutput struct {
	Body SessionBody
}

type LogoutOutput struct {
	Status    int
	SetCookie http.Cookie `header:"Set-Cookie"`
}

func registerAuth(api huma.API, authentication AuthService) {
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
		result, err := authentication.Login(ctx, input.Body.Email, input.Body.Password)
		if errors.Is(err, authdomain.ErrInvalidCredentials) {
			return nil, huma.Error401Unauthorized("E-mail ou senha inválidos")
		}
		if err != nil {
			return nil, huma.Error503ServiceUnavailable("Autenticação temporariamente indisponível")
		}
		return &LoginOutput{
			SetCookie: activeSessionCookie(result.Token, result.ExpiresAt),
			Body:      SessionBody{User: result.User},
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-auth-session",
		Method:      http.MethodGet,
		Path:        "/api/v1/auth/session",
		Summary:     "Describe the current Etpeto session",
		Tags:        []string{"Authentication"},
	}, func(ctx context.Context, input *SessionInput) (*SessionOutput, error) {
		if authentication == nil {
			return nil, huma.Error503ServiceUnavailable("Autenticação temporariamente indisponível")
		}
		user, err := authentication.Authenticate(ctx, input.Session)
		if errors.Is(err, authdomain.ErrInvalidSession) {
			return nil, huma.Error401Unauthorized("Sessão inválida ou expirada")
		}
		if err != nil {
			return nil, huma.Error503ServiceUnavailable("Autenticação temporariamente indisponível")
		}
		return &SessionOutput{Body: SessionBody{User: user}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "logout",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/logout",
		Summary:     "Revoke the current Etpeto session",
		Tags:        []string{"Authentication"},
	}, func(ctx context.Context, input *SessionInput) (*LogoutOutput, error) {
		if authentication == nil {
			return nil, huma.Error503ServiceUnavailable("Autenticação temporariamente indisponível")
		}
		if err := authentication.Logout(ctx, input.Session); err != nil {
			return nil, huma.Error503ServiceUnavailable("Autenticação temporariamente indisponível")
		}
		return &LogoutOutput{
			Status:    http.StatusNoContent,
			SetCookie: expiredSessionCookie(),
		}, nil
	})
}

func activeSessionCookie(token string, expiresAt time.Time) http.Cookie {
	return http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func expiredSessionCookie() http.Cookie {
	return http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(1, 0).UTC(),
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}
