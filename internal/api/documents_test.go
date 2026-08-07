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

	"github.com/etpeto/etpeto/internal/document"
	"github.com/etpeto/etpeto/internal/renderclient"
	"github.com/etpeto/etpeto/internal/rendercontract"
	templatedomain "github.com/etpeto/etpeto/internal/template"
)

type documentServiceStub struct {
	summaries           []document.Summary
	stored              document.Stored
	err                 error
	savedContent        document.Document
	savedExpects        int
	organization        string
	createdFromRevision string
}

// templateServiceStub publishes one revision of the default script, which is
// what a document is created from.
type templateServiceStub struct {
	published []templatedomain.PublishedRevision
	err       error
}

func newTemplateServiceStub() *templateServiceStub {
	return &templateServiceStub{published: []templatedomain.PublishedRevision{{
		ID:           "revision-1",
		TemplateID:   "template-1",
		TemplateName: "Estudo Técnico Preliminar",
		DocumentType: "etp",
		Revision:     1,
	}}}
}

func (s *templateServiceStub) List(context.Context, string) ([]templatedomain.Summary, error) {
	return nil, s.err
}

func (s *templateServiceStub) Get(context.Context, string, string) (templatedomain.Draft, error) {
	return templatedomain.Draft{}, s.err
}

func (s *templateServiceStub) SaveDraft(
	context.Context, string, string, string, int, templatedomain.Blueprint,
) (templatedomain.Draft, error) {
	return templatedomain.Draft{}, s.err
}

func (s *templateServiceStub) Publish(
	context.Context, string, string, string,
) (templatedomain.PublishedRevision, error) {
	return templatedomain.PublishedRevision{}, s.err
}

func (s *templateServiceStub) ListPublished(
	context.Context, string,
) ([]templatedomain.PublishedRevision, error) {
	return s.published, s.err
}

// Only this call carries the script itself, exactly as the repository does.
func (s *templateServiceStub) FindPublished(
	_ context.Context,
	_ string,
	revisionID string,
) (templatedomain.PublishedRevision, error) {
	for _, revision := range s.published {
		if revision.ID == revisionID {
			revision.Blueprint = templatedomain.DefaultBlueprint()
			return revision, nil
		}
	}
	return templatedomain.PublishedRevision{}, templatedomain.ErrNotFound
}

func (s *documentServiceStub) List(_ context.Context, organizationID string) ([]document.Summary, error) {
	s.organization = organizationID
	return s.summaries, s.err
}

func (s *documentServiceStub) Create(
	_ context.Context,
	organizationID string,
	_ string,
	templateRevisionID string,
	content document.Document,
) (document.Stored, error) {
	s.organization = organizationID
	s.createdFromRevision = templateRevisionID
	if s.err != nil {
		return document.Stored{}, s.err
	}
	return document.Stored{ID: "document-1", Revision: 1, Content: content}, nil
}

func (s *documentServiceStub) Open(_ context.Context, organizationID string, _ string) (document.Stored, error) {
	s.organization = organizationID
	return s.stored, s.err
}

func (s *documentServiceStub) Save(
	_ context.Context,
	organizationID string,
	_ string,
	_ string,
	expectedRevision int,
	content document.Document,
) (document.Stored, error) {
	s.organization = organizationID
	s.savedExpects = expectedRevision
	s.savedContent = content
	if s.err != nil {
		return document.Stored{}, s.err
	}
	return document.Stored{ID: "document-1", Revision: expectedRevision + 1, Content: content}, nil
}

func TestListDocumentsScopesTheQueryToTheSessionOrganization(t *testing.T) {
	documents := &documentServiceStub{summaries: []document.Summary{{ID: "document-1", Title: "ETP"}}}
	handler := documentHandler(documents, nil)

	response := sendDocumentRequest(t, handler, http.MethodGet, "/api/v1/documents", nil, validCredentials())

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	if documents.organization != testOrganizationID {
		t.Fatalf("queried organization = %q, want %q", documents.organization, testOrganizationID)
	}
}

func TestDocumentRoutesRejectAnonymousAccess(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{name: "list", method: http.MethodGet, path: "/api/v1/documents"},
		{name: "open", method: http.MethodGet, path: "/api/v1/documents/document-1"},
		{
			name:   "create",
			method: http.MethodPost,
			path:   "/api/v1/documents",
			body:   map[string]string{"title": "ETP"},
		},
		{name: "preview", method: http.MethodPost, path: "/api/v1/documents/document-1/preview"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			documents := &documentServiceStub{}
			handler := documentHandler(documents, nil)
			response := sendDocumentRequest(t, handler, test.method, test.path, test.body, credentials{})

			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
			}
			if documents.organization != "" {
				t.Fatal("an anonymous request reached the document service")
			}
		})
	}
}

func TestMutatingDocumentRoutesRequireTheCSRFToken(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{
			name:   "create",
			method: http.MethodPost,
			path:   "/api/v1/documents",
			body:   map[string]string{"title": "ETP"},
		},
		{
			name:   "save",
			method: http.MethodPut,
			path:   "/api/v1/documents/document-1",
			body:   map[string]any{"expectedRevision": 1, "content": validDocument(t)},
		},
		{name: "preview", method: http.MethodPost, path: "/api/v1/documents/document-1/preview"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			documents := &documentServiceStub{}
			handler := documentHandler(documents, nil)
			response := sendDocumentRequest(
				t,
				handler,
				test.method,
				test.path,
				test.body,
				credentials{session: validSessionToken},
			)

			if response.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusForbidden, response.Body.String())
			}
			if documents.organization != "" {
				t.Fatal("a request without a CSRF token reached the document service")
			}
		})
	}
}

func TestSaveDocumentForwardsTheExpectedRevision(t *testing.T) {
	documents := &documentServiceStub{}
	handler := documentHandler(documents, nil)
	body := map[string]any{"expectedRevision": 3, "content": validDocument(t)}

	response := sendDocumentRequest(t, handler, http.MethodPut, "/api/v1/documents/document-1", body, validCredentials())

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	if documents.savedExpects != 3 {
		t.Fatalf("expected revision = %d, want 3", documents.savedExpects)
	}
	if documents.savedContent.Title != "Aquisição de notebooks" {
		t.Fatalf("saved content = %#v", documents.savedContent)
	}
}

func TestSaveDocumentReportsAConflictInsteadOfOverwriting(t *testing.T) {
	documents := &documentServiceStub{err: document.ErrRevisionConflict}
	handler := documentHandler(documents, nil)
	body := map[string]any{"expectedRevision": 1, "content": validDocument(t)}

	response := sendDocumentRequest(t, handler, http.MethodPut, "/api/v1/documents/document-1", body, validCredentials())

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusConflict, response.Body.String())
	}
}

func TestOpenDocumentReportsMissingDocumentsAsNotFound(t *testing.T) {
	documents := &documentServiceStub{err: document.ErrNotFound}
	handler := documentHandler(documents, nil)

	response := sendDocumentRequest(t, handler, http.MethodGet, "/api/v1/documents/document-1", nil, validCredentials())

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestDocumentPreviewRendersThePersistedRevision(t *testing.T) {
	stored := document.Stored{ID: "document-1", Revision: 2, Content: validDocument(t)}
	renderer := &previewRendererStub{pdf: renderclient.PDF{Bytes: []byte("%PDF stored"), ETag: `"stored"`}}
	documents := &documentServiceStub{stored: stored}
	handler := documentHandler(documents, renderer)

	response := sendDocumentRequest(
		t,
		handler,
		http.MethodPost,
		"/api/v1/documents/document-1/preview",
		nil,
		validCredentials(),
	)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	if response.Body.String() != "%PDF stored" {
		t.Fatalf("body = %q", response.Body.String())
	}
	if renderer.request.Title != "Aquisição de notebooks" {
		t.Fatalf("renderer received = %#v", renderer.request)
	}
	if renderer.request.TemplateRevisionID != defaultTemplateRevisionID {
		t.Fatalf("template revision = %q", renderer.request.TemplateRevisionID)
	}
}

func TestDocumentPreviewHidesRendererFailure(t *testing.T) {
	renderer := &previewRendererStub{err: errors.New("secret renderer endpoint")}
	documents := &documentServiceStub{stored: document.Stored{ID: "document-1", Content: validDocument(t)}}
	handler := documentHandler(documents, renderer)

	response := sendDocumentRequest(
		t,
		handler,
		http.MethodPost,
		"/api/v1/documents/document-1/preview",
		nil,
		validCredentials(),
	)

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

func TestDocumentPreviewRefusesContentTheRendererWouldReject(t *testing.T) {
	broken := validDocument(t)
	broken.Title = ""
	renderer := &previewRendererStub{}
	documents := &documentServiceStub{stored: document.Stored{ID: "document-1", Content: broken}}
	handler := documentHandler(documents, renderer)

	response := sendDocumentRequest(
		t,
		handler,
		http.MethodPost,
		"/api/v1/documents/document-1/preview",
		nil,
		validCredentials(),
	)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnprocessableEntity)
	}
	if renderer.request.Title != "" {
		t.Fatal("unrenderable content reached the renderer")
	}
}

type previewRendererStub struct {
	pdf     renderclient.PDF
	err     error
	request rendercontract.Request
}

func (r *previewRendererStub) Render(
	_ context.Context,
	request rendercontract.Request,
) (renderclient.PDF, error) {
	r.request = request
	return r.pdf, r.err
}

func documentHandler(documents DocumentService, renderer PreviewRenderer) http.Handler {
	return NewRouter(Dependencies{
		Readiness: readyStub{},
		Renderer:  renderer,
		Auth:      &authStub{authUser: authenticatedTestUser()},
		Documents: documents,
		Templates: newTemplateServiceStub(),
	})
}

func sendDocumentRequest(
	t *testing.T,
	handler http.Handler,
	method string,
	path string,
	body any,
	given credentials,
) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	} else {
		reader = bytes.NewReader(nil)
	}
	request := httptest.NewRequest(method, path, reader)
	request.Header.Set("Content-Type", "application/json")
	given.apply(request)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func validDocument(t *testing.T) document.Document {
	t.Helper()
	created, err := document.New("Aquisição de notebooks")
	if err != nil {
		t.Fatal(err)
	}
	return created
}
