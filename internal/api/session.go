package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	authdomain "github.com/etpeto/etpeto/internal/auth"
)

const (
	sessionCookieName = "__Host-etpeto_session"
	csrfCookieName    = "__Host-etpeto_csrf"
	csrfHeaderName    = "X-Etpeto-Csrf"
)

// guard resolves the opaque session cookie and, for mutating routes, the CSRF
// token that the browser echoes back in a request header. The session cookie is
// HttpOnly, so a forged cross-site request can carry the cookie but never the
// header; the header value is checked against the hash stored with the session.
type guard struct {
	authentication AuthService
}

func (g guard) authenticate(ctx context.Context, token string) (authdomain.AuthenticatedUser, error) {
	if g.authentication == nil {
		return authdomain.AuthenticatedUser{}, huma.Error503ServiceUnavailable(
			"Autenticação temporariamente indisponível",
		)
	}
	if token == "" {
		return authdomain.AuthenticatedUser{}, huma.Error401Unauthorized("Sessão inválida ou expirada")
	}
	user, err := g.authentication.Authenticate(ctx, token)
	if errors.Is(err, authdomain.ErrInvalidSession) {
		return authdomain.AuthenticatedUser{}, huma.Error401Unauthorized("Sessão inválida ou expirada")
	}
	if err != nil {
		return authdomain.AuthenticatedUser{}, huma.Error503ServiceUnavailable(
			"Autenticação temporariamente indisponível",
		)
	}
	return user, nil
}

// authorizeMutation authenticates the session and requires a CSRF token bound to
// it. Every mutating authenticated route must go through this method.
func (g guard) authorizeMutation(
	ctx context.Context,
	token string,
	csrfToken string,
) (authdomain.AuthenticatedUser, error) {
	user, err := g.authenticate(ctx, token)
	if err != nil {
		return authdomain.AuthenticatedUser{}, err
	}
	if !authdomain.MatchesTokenHash(user.CSRFHash, csrfToken) {
		return authdomain.AuthenticatedUser{}, huma.Error403Forbidden("Token CSRF ausente ou inválido")
	}
	return user, nil
}

// authorizeRoles authenticates the session, requires the CSRF token and then
// checks the membership role. The role check lives here, in the use case, and
// not only in the interface: hiding a button is a courtesy, refusing the
// request is the control.
func (g guard) authorizeRoles(
	ctx context.Context,
	token string,
	csrfToken string,
	roles ...string,
) (authdomain.AuthenticatedUser, error) {
	user, err := g.authorizeMutation(ctx, token, csrfToken)
	if err != nil {
		return authdomain.AuthenticatedUser{}, err
	}
	for _, role := range roles {
		if user.Role == role {
			return user, nil
		}
	}
	return authdomain.AuthenticatedUser{}, huma.Error403Forbidden(
		"Seu papel não permite esta ação",
	)
}

// requireRoles is the read-only counterpart: no CSRF, because nothing changes.
func (g guard) requireRoles(
	ctx context.Context,
	token string,
	roles ...string,
) (authdomain.AuthenticatedUser, error) {
	user, err := g.authenticate(ctx, token)
	if err != nil {
		return authdomain.AuthenticatedUser{}, err
	}
	for _, role := range roles {
		if user.Role == role {
			return user, nil
		}
	}
	return authdomain.AuthenticatedUser{}, huma.Error403Forbidden(
		"Seu papel não permite esta ação",
	)
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

// activeCSRFCookie is deliberately readable by scripts: the frontend reads it to
// populate the CSRF header after a reload. Confidentiality is not what protects
// it — the same-origin policy is, since a cross-site page cannot read it.
func activeCSRFCookie(token string, expiresAt time.Time) http.Cookie {
	return http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		Secure:   true,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	}
}

func expiredSessionCookie() http.Cookie {
	return expiredCookie(sessionCookieName, true)
}

func expiredCSRFCookie() http.Cookie {
	return expiredCookie(csrfCookieName, false)
}

func expiredCookie(name string, httpOnly bool) http.Cookie {
	return http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(1, 0).UTC(),
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: httpOnly,
		SameSite: http.SameSiteLaxMode,
	}
}
