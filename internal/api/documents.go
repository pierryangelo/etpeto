package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/etpeto/etpeto/internal/document"
	"github.com/etpeto/etpeto/internal/renderclient"
	"github.com/etpeto/etpeto/internal/rendercontract"
)

// PreviewRenderer produces the official PDF. The API is the only caller of the
// renderer, and it only ever sends the closed, versioned contract.
type PreviewRenderer interface {
	Render(context.Context, rendercontract.Request) (renderclient.PDF, error)
}

type PreviewOutput struct {
	ContentType string `header:"Content-Type"`
	ETag        string `header:"ETag"`
	Body        []byte
}

// DocumentService is the application boundary used by the HTTP layer. It always
// receives the organization of the authenticated session.
type DocumentService interface {
	List(ctx context.Context, organizationID string) ([]document.Summary, error)
	Create(ctx context.Context, organizationID string, authorID string, title string) (document.Stored, error)
	Open(ctx context.Context, organizationID string, documentID string) (document.Stored, error)
	Save(
		ctx context.Context,
		organizationID string,
		authorID string,
		documentID string,
		expectedRevision int,
		content document.Document,
	) (document.Stored, error)
}

type DocumentListOutput struct {
	Body struct {
		Documents []document.Summary `json:"documents"`
	}
}

type CreateDocumentInput struct {
	Session string `cookie:"__Host-etpeto_session"`
	CSRF    string `header:"X-Etpeto-Csrf"`
	Body    struct {
		Title string `json:"title" maxLength:"300"`
	}
}

type DocumentOutput struct {
	Body document.Stored
}

type ListDocumentsInput struct {
	Session string `cookie:"__Host-etpeto_session"`
}

type OpenDocumentInput struct {
	Session string `cookie:"__Host-etpeto_session"`
	ID      string `path:"id"`
}

type SaveDocumentInput struct {
	Session string `cookie:"__Host-etpeto_session"`
	CSRF    string `header:"X-Etpeto-Csrf"`
	ID      string `path:"id"`
	Body    struct {
		ExpectedRevision int               `json:"expectedRevision"`
		Content          document.Document `json:"content"`
	}
}

type DocumentPreviewInput struct {
	Session string `cookie:"__Host-etpeto_session"`
	CSRF    string `header:"X-Etpeto-Csrf"`
	ID      string `path:"id"`
}

func registerDocuments(
	api huma.API,
	documents DocumentService,
	renderer PreviewRenderer,
	protected guard,
) {
	huma.Register(api, huma.Operation{
		OperationID: "list-documents",
		Method:      http.MethodGet,
		Path:        "/api/v1/documents",
		Summary:     "List the documents of the current organization",
		Tags:        []string{"Documents"},
	}, func(ctx context.Context, input *ListDocumentsInput) (*DocumentListOutput, error) {
		user, err := protected.authenticate(ctx, input.Session)
		if err != nil {
			return nil, err
		}
		if documents == nil {
			return nil, huma.Error503ServiceUnavailable("Documentos temporariamente indisponíveis")
		}
		summaries, err := documents.List(ctx, user.OrganizationID)
		if err != nil {
			return nil, documentError(err)
		}
		output := &DocumentListOutput{}
		output.Body.Documents = summaries
		if output.Body.Documents == nil {
			output.Body.Documents = []document.Summary{}
		}
		return output, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "create-document",
		Method:        http.MethodPost,
		Path:          "/api/v1/documents",
		Summary:       "Create a document from the default outline",
		Tags:          []string{"Documents"},
		DefaultStatus: http.StatusCreated,
		MaxBodyBytes:  4 << 10,
	}, func(ctx context.Context, input *CreateDocumentInput) (*DocumentOutput, error) {
		user, err := protected.authorizeMutation(ctx, input.Session, input.CSRF)
		if err != nil {
			return nil, err
		}
		if documents == nil {
			return nil, huma.Error503ServiceUnavailable("Documentos temporariamente indisponíveis")
		}
		created, err := documents.Create(ctx, user.OrganizationID, user.ID, input.Body.Title)
		if err != nil {
			return nil, documentError(err)
		}
		return &DocumentOutput{Body: created}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "open-document",
		Method:      http.MethodGet,
		Path:        "/api/v1/documents/{id}",
		Summary:     "Open a document and its current revision",
		Tags:        []string{"Documents"},
	}, func(ctx context.Context, input *OpenDocumentInput) (*DocumentOutput, error) {
		user, err := protected.authenticate(ctx, input.Session)
		if err != nil {
			return nil, err
		}
		if documents == nil {
			return nil, huma.Error503ServiceUnavailable("Documentos temporariamente indisponíveis")
		}
		opened, err := documents.Open(ctx, user.OrganizationID, input.ID)
		if err != nil {
			return nil, documentError(err)
		}
		return &DocumentOutput{Body: opened}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:  "save-document",
		Method:       http.MethodPut,
		Path:         "/api/v1/documents/{id}",
		Summary:      "Save the next working revision of a document",
		Tags:         []string{"Documents"},
		MaxBodyBytes: 2 << 20,
	}, func(ctx context.Context, input *SaveDocumentInput) (*DocumentOutput, error) {
		user, err := protected.authorizeMutation(ctx, input.Session, input.CSRF)
		if err != nil {
			return nil, err
		}
		if documents == nil {
			return nil, huma.Error503ServiceUnavailable("Documentos temporariamente indisponíveis")
		}
		saved, err := documents.Save(
			ctx,
			user.OrganizationID,
			user.ID,
			input.ID,
			input.Body.ExpectedRevision,
			input.Body.Content,
		)
		if err != nil {
			return nil, documentError(err)
		}
		return &DocumentOutput{Body: saved}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "preview-document",
		Method:      http.MethodPost,
		Path:        "/api/v1/documents/{id}/preview",
		Summary:     "Render the official PDF preview of a stored document",
		Tags:        []string{"Documents"},
		Responses: map[string]*huma.Response{
			"200": {
				Description: "Rendered PDF preview",
				Content:     map[string]*huma.MediaType{"application/pdf": {}},
			},
		},
	}, func(ctx context.Context, input *DocumentPreviewInput) (*PreviewOutput, error) {
		user, err := protected.authorizeMutation(ctx, input.Session, input.CSRF)
		if err != nil {
			return nil, err
		}
		if documents == nil || renderer == nil {
			return nil, huma.Error503ServiceUnavailable("Renderização temporariamente indisponível")
		}
		opened, err := documents.Open(ctx, user.OrganizationID, input.ID)
		if err != nil {
			return nil, documentError(err)
		}
		// The preview always renders what is persisted, so the PDF can never
		// show content that was never saved.
		request := document.Normalize(opened.Content, defaultTemplateRevisionID)
		if err := request.Validate(); err != nil {
			return nil, huma.Error422UnprocessableEntity("Documento não pode ser renderizado")
		}
		pdf, err := renderer.Render(ctx, request)
		if err != nil {
			return nil, huma.Error503ServiceUnavailable("Renderização temporariamente indisponível")
		}
		return &PreviewOutput{
			ContentType: "application/pdf",
			ETag:        pdf.ETag,
			Body:        pdf.Bytes,
		}, nil
	})
}

// defaultTemplateRevisionID stands in for a published template revision until
// configurable, versioned scripts exist.
const defaultTemplateRevisionID = "etp-default-v1"

func documentError(err error) error {
	switch {
	case errors.Is(err, document.ErrNotFound):
		return huma.Error404NotFound("Documento não encontrado")
	case errors.Is(err, document.ErrRevisionConflict):
		return huma.Error409Conflict("O documento foi alterado por outra sessão")
	case errors.Is(err, document.ErrInvalidDocument):
		return huma.Error422UnprocessableEntity("Conteúdo do documento é inválido")
	default:
		return huma.Error503ServiceUnavailable("Documentos temporariamente indisponíveis")
	}
}
