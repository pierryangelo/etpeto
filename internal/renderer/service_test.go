package renderer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/etpeto/etpeto/internal/rendercontract"
)

type runnerStub struct {
	pdf    []byte
	err    error
	source string
}

func (r *runnerStub) Render(_ context.Context, source string) ([]byte, error) {
	r.source = source
	return r.pdf, r.err
}

func TestRenderReturnsPDFAndETag(t *testing.T) {
	runner := &runnerStub{pdf: []byte("%PDF-1.7 fixture")}
	handler := NewHandler(runner)
	body, err := json.Marshal(validRenderRequest())
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/internal/render", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/pdf" {
		t.Fatalf("Content-Type = %q", contentType)
	}
	if etag := response.Header().Get("ETag"); !strings.HasPrefix(etag, `"`) || !strings.HasSuffix(etag, `"`) {
		t.Fatalf("ETag = %q, want quoted hash", etag)
	}
	if response.Body.String() != string(runner.pdf) {
		t.Fatalf("body = %q, want PDF fixture", response.Body.String())
	}
	if !strings.Contains(runner.source, "Necessidade") {
		t.Fatal("normalized request was not converted to Typst source")
	}
}

func TestRenderHidesRunnerDetails(t *testing.T) {
	runner := &runnerStub{err: errors.New("secret Typst stderr with document content")}
	handler := NewHandler(runner)
	body, _ := json.Marshal(validRenderRequest())
	request := httptest.NewRequest(http.MethodPost, "/internal/render", bytes.NewReader(body))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if strings.Contains(response.Body.String(), "secret") {
		t.Fatalf("runner detail leaked: %s", response.Body.String())
	}
	if response.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("Content-Type = %q", response.Header().Get("Content-Type"))
	}
}

func TestRenderRejectsBodyLargerThanTwoMiB(t *testing.T) {
	handler := NewHandler(&runnerStub{})
	request := httptest.NewRequest(
		http.MethodPost,
		"/internal/render",
		strings.NewReader(strings.Repeat("x", maxRequestBytes+1)),
	)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
}

func validRenderRequest() rendercontract.Request {
	return rendercontract.Request{
		SchemaVersion:      1,
		TemplateRevisionID: "template-v1",
		Title:              "ETP",
		Sections: []rendercontract.Section{{
			Title:      "Necessidade",
			Paragraphs: []string{"Texto"},
		}},
	}
}
