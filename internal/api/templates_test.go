package api

import (
	"net/http"
	"testing"

	authdomain "github.com/etpeto/etpeto/internal/auth"
	templatedomain "github.com/etpeto/etpeto/internal/template"
)

func templateHandler(user authdomain.AuthenticatedUser) (http.Handler, *templateServiceStub) {
	templates := newTemplateServiceStub()
	return NewRouter(Dependencies{
		Readiness: readyStub{},
		Auth:      &authStub{authUser: user},
		Templates: templates,
		Documents: &documentServiceStub{},
	}), templates
}

func userWithRole(role string) authdomain.AuthenticatedUser {
	user := authenticatedTestUser()
	user.Role = role
	return user
}

// The role is checked in the use case, not only in the interface: hiding a
// button is a courtesy, refusing the request is the control.
func TestScriptRoutesRefuseAnAuthorEvenWithAValidSession(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{name: "list", method: http.MethodGet, path: "/api/v1/templates"},
		{name: "open draft", method: http.MethodGet, path: "/api/v1/templates/template-1"},
		{
			name:   "save draft",
			method: http.MethodPut,
			path:   "/api/v1/templates/template-1/draft",
			body: map[string]any{
				"expectedVersion": 1,
				"blueprint":       templatedomain.DefaultBlueprint(),
			},
		},
		{
			name:   "publish",
			method: http.MethodPost,
			path:   "/api/v1/templates/template-1/revisions",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, _ := templateHandler(userWithRole(authdomain.RoleAuthor))
			response := sendDocumentRequest(t, handler, test.method, test.path, test.body, validCredentials())

			if response.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d; body=%s",
					response.Code, http.StatusForbidden, response.Body.String())
			}
		})
	}
}

func TestScriptRoutesAcceptATemplateManager(t *testing.T) {
	handler, _ := templateHandler(userWithRole(authdomain.RoleTemplateManager))

	response := sendDocumentRequest(t, handler, http.MethodGet, "/api/v1/templates", nil, validCredentials())

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestScriptRoutesRejectAnonymousAccess(t *testing.T) {
	handler, _ := templateHandler(userWithRole(authdomain.RoleTemplateManager))

	response := sendDocumentRequest(t, handler, http.MethodGet, "/api/v1/templates", nil, credentials{})

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestSavingADraftRequiresTheCSRFToken(t *testing.T) {
	handler, _ := templateHandler(userWithRole(authdomain.RoleTemplateManager))
	body := map[string]any{
		"expectedVersion": 1,
		"blueprint":       templatedomain.DefaultBlueprint(),
	}

	response := sendDocumentRequest(
		t, handler, http.MethodPut, "/api/v1/templates/template-1/draft", body,
		credentials{session: validSessionToken},
	)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

// Authors do need the published list: it is how they pick a script when
// creating a document.
func TestPublishedScriptsAreReadableByAnyAuthor(t *testing.T) {
	handler, _ := templateHandler(userWithRole(authdomain.RoleAuthor))

	response := sendDocumentRequest(
		t, handler, http.MethodGet, "/api/v1/templates/published", nil, validCredentials(),
	)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestCreatingADocumentBindsItToAPublishedRevision(t *testing.T) {
	documents := &documentServiceStub{}
	handler := NewRouter(Dependencies{
		Readiness: readyStub{},
		Auth:      &authStub{authUser: userWithRole(authdomain.RoleAuthor)},
		Documents: documents,
		Templates: newTemplateServiceStub(),
	})

	response := sendDocumentRequest(
		t, handler, http.MethodPost, "/api/v1/documents",
		map[string]string{"title": "Aquisição de notebooks"}, validCredentials(),
	)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusCreated, response.Body.String())
	}
	if documents.createdFromRevision != "revision-1" {
		t.Fatalf("document revision binding = %q", documents.createdFromRevision)
	}
	// The outline comes from the published script, not from a list in the code.
	if len(documents.savedContent.Sections) != 0 {
		t.Fatal("creating a document should not have gone through save")
	}
}

func TestCreatingADocumentFailsWhenNoScriptIsPublished(t *testing.T) {
	templates := newTemplateServiceStub()
	templates.published = nil
	handler := NewRouter(Dependencies{
		Readiness: readyStub{},
		Auth:      &authStub{authUser: userWithRole(authdomain.RoleAuthor)},
		Documents: &documentServiceStub{},
		Templates: templates,
	})

	response := sendDocumentRequest(
		t, handler, http.MethodPost, "/api/v1/documents",
		map[string]string{"title": "Aquisição de notebooks"}, validCredentials(),
	)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d; body=%s",
			response.Code, http.StatusUnprocessableEntity, response.Body.String())
	}
}
