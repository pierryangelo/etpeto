package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type readyStub struct {
	err error
}

func (s readyStub) Ready(context.Context) error {
	return s.err
}

func TestLivenessReportsOK(t *testing.T) {
	handler := NewRouter(Dependencies{Readiness: readyStub{}})

	response := request(t, handler, "/livez")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	assertJSONStatus(t, response, "ok")
}

func TestReadinessReportsUnavailableWithoutLeakingDependencyError(t *testing.T) {
	handler := NewRouter(Dependencies{Readiness: readyStub{err: errors.New("dial postgres at secret-host")}})

	response := request(t, handler, "/readyz")

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if strings.Contains(response.Body.String(), "secret-host") {
		t.Fatalf("response leaked dependency detail: %s", response.Body.String())
	}
	assertJSONStatus(t, response, "unavailable")
}

func TestServiceInfoIdentifiesEtpeto(t *testing.T) {
	handler := NewRouter(Dependencies{Readiness: readyStub{}})

	response := request(t, handler, "/api/v1/service-info")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var body struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Name != "Etpeto" || body.Version != "0.1.0-dev" {
		t.Fatalf("service info = %#v", body)
	}
}

func TestOpenAPIContainsPublicOperations(t *testing.T) {
	handler := NewRouter(Dependencies{Readiness: readyStub{}})

	response := request(t, handler, "/openapi.json")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	body := response.Body.String()
	for _, path := range []string{
		"/livez",
		"/readyz",
		"/api/v1/service-info",
		"/api/v1/auth/local/login",
		"/api/v1/auth/session",
		"/api/v1/auth/logout",
		"/api/v1/documents",
		"/api/v1/documents/{id}",
		"/api/v1/documents/{id}/preview",
	} {
		if !strings.Contains(body, path) {
			t.Errorf("OpenAPI does not contain %s", path)
		}
	}
}

func request(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func assertJSONStatus(t *testing.T, response *httptest.ResponseRecorder, want string) {
	t.Helper()
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != want {
		t.Fatalf("status body = %q, want %q", body.Status, want)
	}
}
