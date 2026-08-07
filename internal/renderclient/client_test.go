package renderclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/etpeto/etpeto/internal/rendercontract"
)

func TestClientRenderSendsAuthenticatedContractAndReturnsPDF(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/internal/render" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		if request.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("Content-Type = %q", request.Header.Get("Content-Type"))
		}
		if request.Header.Get("X-Etpeto-Renderer-Token") != "renderer-secret" {
			t.Fatal("renderer token was not sent")
		}
		var body rendercontract.Request
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body.Title != "ETP de notebooks" {
			t.Fatalf("title = %q", body.Title)
		}
		response.Header().Set("Content-Type", "application/pdf")
		response.Header().Set("ETag", `"pdf-hash"`)
		_, _ = response.Write([]byte("%PDF-1.7 fixture"))
	}))
	defer server.Close()

	client := Client{BaseURL: server.URL, Token: "renderer-secret", HTTP: server.Client()}
	pdf, err := client.Render(context.Background(), validClientRequest())
	if err != nil {
		t.Fatal(err)
	}
	if string(pdf.Bytes) != "%PDF-1.7 fixture" || pdf.ETag != `"pdf-hash"` {
		t.Fatalf("PDF = %#v", pdf)
	}
}

func TestClientRenderRejectsNonPDFWithoutLeakingBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "text/plain")
		response.WriteHeader(http.StatusServiceUnavailable)
		_, _ = response.Write([]byte("secret renderer failure"))
	}))
	defer server.Close()

	client := Client{BaseURL: server.URL, Token: "renderer-secret", HTTP: server.Client()}
	_, err := client.Render(context.Background(), validClientRequest())
	if err == nil {
		t.Fatal("Render() succeeded for renderer failure")
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("renderer body leaked through error: %v", err)
	}
}

func TestClientRenderRejectsOversizedPDFBeforeReadingIt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/pdf")
		response.Header().Set("Content-Length", strconv.Itoa(maxPDFBytes+1))
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := Client{BaseURL: server.URL, Token: "renderer-secret", HTTP: server.Client()}
	_, err := client.Render(context.Background(), validClientRequest())
	if err == nil || !strings.Contains(err.Error(), "size limit") {
		t.Fatalf("Render() error = %v, want size limit", err)
	}
}

func TestClientRenderRejectsSuccessfulNonPDF(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "text/html")
		_, _ = response.Write([]byte("not a PDF"))
	}))
	defer server.Close()

	client := Client{BaseURL: server.URL, Token: "renderer-secret", HTTP: server.Client()}
	_, err := client.Render(context.Background(), validClientRequest())
	if err == nil || !strings.Contains(err.Error(), "content type") {
		t.Fatalf("Render() error = %v, want content type error", err)
	}
}

func TestClientRenderDoesNotForwardTokenThroughRedirect(t *testing.T) {
	redirectFollowed := false
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/elsewhere" {
			redirectFollowed = true
			response.Header().Set("Content-Type", "application/pdf")
			_, _ = response.Write([]byte("%PDF"))
			return
		}
		http.Redirect(response, request, "/elsewhere", http.StatusTemporaryRedirect)
	}))
	defer server.Close()

	client := Client{BaseURL: server.URL, Token: "renderer-secret", HTTP: server.Client()}
	_, err := client.Render(context.Background(), validClientRequest())
	if err == nil {
		t.Fatal("Render() followed renderer redirect")
	}
	if redirectFollowed {
		t.Fatal("redirect target received a request that could carry the renderer token")
	}
}

func validClientRequest() rendercontract.Request {
	return rendercontract.Request{
		SchemaVersion:      1,
		TemplateRevisionID: "template-v1",
		Title:              "ETP de notebooks",
		Sections:           []rendercontract.Section{{Title: "Necessidade", Level: 1, Paragraphs: []string{"Texto"}}},
	}
}
