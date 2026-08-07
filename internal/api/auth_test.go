package api

import (
	"bytes"
	"context"
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
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %#v", cookies)
	}
	cookie := cookies[0]
	if cookie.Name != sessionCookieName || cookie.Value != "opaque-session-token" {
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

func TestLogoutRevokesCurrentSessionAndExpiresCookie(t *testing.T) {
	authentication := &authStub{}
	handler := NewRouter(Dependencies{Readiness: readyStub{}, Auth: authentication})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "current-token"})
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if authentication.logoutToken != "current-token" {
		t.Fatalf("revoked token = %q", authentication.logoutToken)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge >= 0 || cookies[0].Value != "" {
		t.Fatalf("logout cookie = %#v", cookies)
	}
}

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
