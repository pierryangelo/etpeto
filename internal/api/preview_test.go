package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/etpeto/etpeto/internal/renderclient"
	"github.com/etpeto/etpeto/internal/rendercontract"
)

type previewRendererStub struct {
	pdf     renderclient.PDF
	err     error
	request rendercontract.Request
}

func (r *previewRendererStub) Render(_ context.Context, request rendercontract.Request) (renderclient.PDF, error) {
	r.request = request
	return r.pdf, r.err
}

func TestPreviewReturnsRendererPDF(t *testing.T) {
	renderer := &previewRendererStub{pdf: renderclient.PDF{Bytes: []byte("%PDF fixture"), ETag: `"hash"`}}
	handler := previewHandler(renderer)
	response := postPreview(t, handler, validPreviewRequest(), validCredentials())

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	if response.Header().Get("Content-Type") != "application/pdf" {
		t.Fatalf("Content-Type = %q", response.Header().Get("Content-Type"))
	}
	if response.Header().Get("ETag") != `"hash"` || response.Body.String() != "%PDF fixture" {
		t.Fatalf("response headers=%v body=%q", response.Header(), response.Body.String())
	}
	if renderer.request.Title != "ETP de notebooks" {
		t.Fatalf("renderer request = %#v", renderer.request)
	}
}

func TestPreviewRejectsAnonymousRequest(t *testing.T) {
	renderer := &previewRendererStub{}
	response := postPreview(t, previewHandler(renderer), validPreviewRequest(), credentials{})

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusUnauthorized, response.Body.String())
	}
	if renderer.request.Title != "" {
		t.Fatal("anonymous request reached the renderer")
	}
}

func TestPreviewRejectsMissingOrForgedCSRFToken(t *testing.T) {
	tests := []struct {
		name  string
		given credentials
	}{
		{name: "missing token", given: credentials{session: validSessionToken}},
		{name: "forged token", given: credentials{session: validSessionToken, csrf: forgedCSRFToken}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			renderer := &previewRendererStub{}
			response := postPreview(t, previewHandler(renderer), validPreviewRequest(), test.given)

			if response.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusForbidden, response.Body.String())
			}
			if renderer.request.Title != "" {
				t.Fatal("request without a valid CSRF token reached the renderer")
			}
		})
	}
}

func TestPreviewRejectsInvalidContract(t *testing.T) {
	renderer := &previewRendererStub{}
	handler := previewHandler(renderer)
	request := validPreviewRequest()
	request.SchemaVersion = 99
	response := postPreview(t, handler, request, validCredentials())

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusUnprocessableEntity, response.Body.String())
	}
	if renderer.request.Title != "" {
		t.Fatal("invalid request reached renderer")
	}
}

func TestPreviewHidesRendererFailure(t *testing.T) {
	renderer := &previewRendererStub{err: errors.New("secret renderer endpoint")}
	handler := previewHandler(renderer)
	response := postPreview(t, handler, validPreviewRequest(), validCredentials())

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if strings.Contains(response.Body.String(), "secret") {
		t.Fatalf("renderer detail leaked: %s", response.Body.String())
	}
	if !strings.HasPrefix(response.Header().Get("Content-Type"), "application/problem+json") {
		t.Fatalf("Content-Type = %q", response.Header().Get("Content-Type"))
	}
}

func previewHandler(renderer PreviewRenderer) http.Handler {
	return NewRouter(Dependencies{
		Readiness: readyStub{},
		Renderer:  renderer,
		Auth:      &authStub{authUser: authenticatedTestUser()},
	})
}

func postPreview(
	t *testing.T,
	handler http.Handler,
	request rendercontract.Request,
	given credentials,
) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	httpRequest := httptest.NewRequest(http.MethodPost, "/api/v1/previews", bytes.NewReader(body))
	httpRequest.Header.Set("Content-Type", "application/json")
	given.apply(httpRequest)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httpRequest)
	return response
}

func validPreviewRequest() rendercontract.Request {
	return rendercontract.Request{
		SchemaVersion:      1,
		TemplateRevisionID: "template-v1",
		Title:              "ETP de notebooks",
		Sections:           []rendercontract.Section{{Title: "Necessidade", Paragraphs: []string{"Texto"}}},
	}
}
