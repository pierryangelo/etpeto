package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authdomain "github.com/etpeto/etpeto/internal/auth"
)

type authStub struct {
	loginResult authdomain.LoginResult
	loginErr    error
	authUser    authdomain.AuthenticatedUser
	authErr     error
	logoutToken string
}

func (s *authStub) Login(context.Context, string, string) (authdomain.LoginResult, error) {
	return s.loginResult, s.loginErr
}

func (s *authStub) Authenticate(context.Context, string) (authdomain.AuthenticatedUser, error) {
	return s.authUser, s.authErr
}

func (s *authStub) Logout(_ context.Context, token string) error {
	s.logoutToken = token
	return nil
}

func TestLocalLoginSetsHardenedSessionCookie(t *testing.T) {
	expiresAt := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	authentication := &authStub{loginResult: authdomain.LoginResult{
		Token:     "opaque-session-token",
		ExpiresAt: expiresAt,
		User: authdomain.AuthenticatedUser{
			ID:          "user-1",
			DisplayName: "Pessoa Autora",
		},
	}}
	handler := NewRouter(Dependencies{Readiness: readyStub{}, Auth: authentication})
	response := postJSON(t, handler, "/api/v1/auth/local/login", map[string]string{
		"email":    "pessoa@orgao.gov.br",
		"password": "senha segura",
	})

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	cookie := cookieNamed(t, response.Result().Cookies(), sessionCookieName)
	if cookie.Value != "opaque-session-token" {
		t.Fatalf("cookie = %#v", cookie)
	}
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" || cookie.Domain != "" {
		t.Fatalf("cookie attributes = %#v", cookie)
	}
}

func TestLocalLoginUsesIdenticalUnauthorizedBody(t *testing.T) {
	var bodies []string
	for _, name := range []string{"missing account", "wrong password"} {
		t.Run(name, func(t *testing.T) {
			handler := NewRouter(Dependencies{
				Readiness: readyStub{},
				Auth:      &authStub{loginErr: authdomain.ErrInvalidCredentials},
			})
			response := postJSON(t, handler, "/api/v1/auth/local/login", map[string]string{
				"email":    "pessoa@orgao.gov.br",
				"password": "incorreta",
			})
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
			}
			bodies = append(bodies, response.Body.String())
		})
	}
	if bodies[0] != bodies[1] {
		t.Fatalf("credential error bodies differ:\n%s\n%s", bodies[0], bodies[1])
	}
}

func TestLocalLoginPublishesReadableCSRFCookie(t *testing.T) {
	authentication := &authStub{loginResult: authdomain.LoginResult{
		Token:     "opaque-session-token",
		CSRFToken: validCSRFToken(),
		ExpiresAt: time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC),
	}}
	handler := NewRouter(Dependencies{Readiness: readyStub{}, Auth: authentication})
	response := postJSON(t, handler, "/api/v1/auth/local/login", map[string]string{
		"email":    "pessoa@orgao.gov.br",
		"password": "senha segura",
	})

	csrf := cookieNamed(t, response.Result().Cookies(), csrfCookieName)
	if csrf.Value != validCSRFToken() {
		t.Fatalf("CSRF cookie value = %q", csrf.Value)
	}
	if csrf.HttpOnly {
		t.Fatal("CSRF cookie is HttpOnly and cannot be echoed back by the browser")
	}
	if !csrf.Secure || csrf.SameSite != http.SameSiteLaxMode || csrf.Path != "/" || csrf.Domain != "" {
		t.Fatalf("CSRF cookie attributes = %#v", csrf)
	}
}

func TestLogoutRevokesCurrentSessionAndExpiresCookies(t *testing.T) {
	authentication := &authStub{authUser: authenticatedTestUser()}
	handler := NewRouter(Dependencies{Readiness: readyStub{}, Auth: authentication})
	response := postLogout(t, handler, validCredentials())

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if authentication.logoutToken != validSessionToken {
		t.Fatalf("revoked token = %q", authentication.logoutToken)
	}
	for _, name := range []string{sessionCookieName, csrfCookieName} {
		cookie := cookieNamed(t, response.Result().Cookies(), name)
		if cookie.MaxAge >= 0 || cookie.Value != "" {
			t.Fatalf("%s was not expired: %#v", name, cookie)
		}
	}
}

func TestLogoutRequiresCSRFTokenWhileTheSessionIsValid(t *testing.T) {
	authentication := &authStub{authUser: authenticatedTestUser()}
	handler := NewRouter(Dependencies{Readiness: readyStub{}, Auth: authentication})
	response := postLogout(t, handler, credentials{session: validSessionToken})

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusForbidden, response.Body.String())
	}
	if authentication.logoutToken != "" {
		t.Fatal("session was revoked without a valid CSRF token")
	}
}

func TestLogoutStaysIdempotentForAnExpiredSession(t *testing.T) {
	authentication := &authStub{authErr: authdomain.ErrInvalidSession}
	handler := NewRouter(Dependencies{Readiness: readyStub{}, Auth: authentication})
	response := postLogout(t, handler, credentials{session: "expired-token"})

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusNoContent, response.Body.String())
	}
	cookie := cookieNamed(t, response.Result().Cookies(), sessionCookieName)
	if cookie.MaxAge >= 0 {
		t.Fatalf("expired session did not clear the cookie: %#v", cookie)
	}
}

func postLogout(t *testing.T, handler http.Handler, given credentials) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	given.apply(request)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func cookieNamed(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %q not found in %#v", name, cookies)
	return nil
}

const validSessionToken = "opaque-session-token"

// credentials models what a browser sends: the session in an HttpOnly cookie
// and the CSRF token echoed back in a request header.
type credentials struct {
	session string
	csrf    string
}

func validCredentials() credentials {
	return credentials{session: validSessionToken, csrf: validCSRFToken()}
}

func (c credentials) apply(request *http.Request) {
	if c.session != "" {
		request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: c.session})
	}
	if c.csrf != "" {
		request.Header.Set(csrfHeaderName, c.csrf)
	}
}

func authenticatedTestUser() authdomain.AuthenticatedUser {
	hash := sha256.Sum256(bytes.Repeat([]byte{0x21}, 32))
	return authdomain.AuthenticatedUser{
		ID:          "00000000-0000-0000-0000-000000000001",
		DisplayName: "Pessoa Autora",
		ExpiresAt:   time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC),
		CSRFHash:    hash[:],
	}
}

func validCSRFToken() string {
	return base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x21}, 32))
}

var forgedCSRFToken = base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x22}, 32))

func postJSON(
	t *testing.T,
	handler http.Handler,
	path string,
	body any,
) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(encoded))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
