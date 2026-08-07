package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	authdomain "github.com/etpeto/etpeto/internal/auth"
	templatedomain "github.com/etpeto/etpeto/internal/template"
)

// Roles allowed to shape the institutional script. Authors read published
// revisions to create a document, but never edit one.
var scriptManagerRoles = []string{
	authdomain.RoleOrganizationAdmin,
	authdomain.RoleTemplateManager,
	authdomain.RoleInstallationAdmin,
}

type TemplateService interface {
	List(ctx context.Context, organizationID string) ([]templatedomain.Summary, error)
	Get(ctx context.Context, organizationID string, templateID string) (templatedomain.Draft, error)
	SaveDraft(
		ctx context.Context,
		organizationID string,
		templateID string,
		authorID string,
		expectedVersion int,
		blueprint templatedomain.Blueprint,
	) (templatedomain.Draft, error)
	Publish(
		ctx context.Context,
		organizationID string,
		templateID string,
		authorID string,
	) (templatedomain.PublishedRevision, error)
	ListPublished(ctx context.Context, organizationID string) ([]templatedomain.PublishedRevision, error)
	// FindPublished returns the revision with its blueprint. The list is
	// deliberately light and carries no script content.
	FindPublished(
		ctx context.Context,
		organizationID string,
		revisionID string,
	) (templatedomain.PublishedRevision, error)
}

type TemplateListOutput struct {
	Body struct {
		Templates []templatedomain.Summary `json:"templates"`
	}
}

type PublishedTemplateListOutput struct {
	Body struct {
		Templates []templatedomain.PublishedRevision `json:"templates"`
	}
}

type TemplateInput struct {
	Session string `cookie:"__Host-etpeto_session"`
	ID      string `path:"id"`
}

type SaveTemplateDraftInput struct {
	Session string `cookie:"__Host-etpeto_session"`
	CSRF    string `header:"X-Etpeto-Csrf"`
	ID      string `path:"id"`
	Body    struct {
		ExpectedVersion int                      `json:"expectedVersion"`
		Blueprint       templatedomain.Blueprint `json:"blueprint"`
	}
}

type PublishTemplateInput struct {
	Session string `cookie:"__Host-etpeto_session"`
	CSRF    string `header:"X-Etpeto-Csrf"`
	ID      string `path:"id"`
}

type TemplateDraftOutput struct {
	Body templatedomain.Draft
}

type PublishedRevisionOutput struct {
	Body templatedomain.PublishedRevision
}

func registerTemplates(api huma.API, templates TemplateService, protected guard) {
	huma.Register(api, huma.Operation{
		OperationID: "list-templates",
		Method:      http.MethodGet,
		Path:        "/api/v1/templates",
		Summary:     "List the scripts of the organization",
		Tags:        []string{"Templates"},
	}, func(ctx context.Context, input *ListDocumentsInput) (*TemplateListOutput, error) {
		user, err := protected.requireRoles(ctx, input.Session, scriptManagerRoles...)
		if err != nil {
			return nil, err
		}
		if templates == nil {
			return nil, huma.Error503ServiceUnavailable("Roteiros temporariamente indisponíveis")
		}
		summaries, err := templates.List(ctx, user.OrganizationID)
		if err != nil {
			return nil, templateError(err)
		}
		output := &TemplateListOutput{}
		output.Body.Templates = summaries
		if output.Body.Templates == nil {
			output.Body.Templates = []templatedomain.Summary{}
		}
		return output, nil
	})

	// Authors need the published list to create a document, so this route is
	// open to any session — it exposes names and revisions, never the draft.
	huma.Register(api, huma.Operation{
		OperationID: "list-published-templates",
		Method:      http.MethodGet,
		Path:        "/api/v1/templates/published",
		Summary:     "List the published script revisions available for new documents",
		Tags:        []string{"Templates"},
	}, func(ctx context.Context, input *ListDocumentsInput) (*PublishedTemplateListOutput, error) {
		user, err := protected.authenticate(ctx, input.Session)
		if err != nil {
			return nil, err
		}
		if templates == nil {
			return nil, huma.Error503ServiceUnavailable("Roteiros temporariamente indisponíveis")
		}
		published, err := templates.ListPublished(ctx, user.OrganizationID)
		if err != nil {
			return nil, templateError(err)
		}
		output := &PublishedTemplateListOutput{}
		output.Body.Templates = published
		if output.Body.Templates == nil {
			output.Body.Templates = []templatedomain.PublishedRevision{}
		}
		return output, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-template-draft",
		Method:      http.MethodGet,
		Path:        "/api/v1/templates/{id}",
		Summary:     "Open the working draft of a script",
		Tags:        []string{"Templates"},
	}, func(ctx context.Context, input *TemplateInput) (*TemplateDraftOutput, error) {
		user, err := protected.requireRoles(ctx, input.Session, scriptManagerRoles...)
		if err != nil {
			return nil, err
		}
		if templates == nil {
			return nil, huma.Error503ServiceUnavailable("Roteiros temporariamente indisponíveis")
		}
		draft, err := templates.Get(ctx, user.OrganizationID, input.ID)
		if err != nil {
			return nil, templateError(err)
		}
		return &TemplateDraftOutput{Body: draft}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:  "save-template-draft",
		Method:       http.MethodPut,
		Path:         "/api/v1/templates/{id}/draft",
		Summary:      "Save the working draft of a script",
		Tags:         []string{"Templates"},
		MaxBodyBytes: 1 << 20,
	}, func(ctx context.Context, input *SaveTemplateDraftInput) (*TemplateDraftOutput, error) {
		user, err := protected.authorizeRoles(ctx, input.Session, input.CSRF, scriptManagerRoles...)
		if err != nil {
			return nil, err
		}
		if templates == nil {
			return nil, huma.Error503ServiceUnavailable("Roteiros temporariamente indisponíveis")
		}
		draft, err := templates.SaveDraft(
			ctx,
			user.OrganizationID,
			input.ID,
			user.ID,
			input.Body.ExpectedVersion,
			input.Body.Blueprint,
		)
		if err != nil {
			return nil, templateError(err)
		}
		return &TemplateDraftOutput{Body: draft}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "publish-template",
		Method:        http.MethodPost,
		Path:          "/api/v1/templates/{id}/revisions",
		Summary:       "Publish the draft as the next immutable revision",
		Tags:          []string{"Templates"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *PublishTemplateInput) (*PublishedRevisionOutput, error) {
		user, err := protected.authorizeRoles(ctx, input.Session, input.CSRF, scriptManagerRoles...)
		if err != nil {
			return nil, err
		}
		if templates == nil {
			return nil, huma.Error503ServiceUnavailable("Roteiros temporariamente indisponíveis")
		}
		published, err := templates.Publish(ctx, user.OrganizationID, input.ID, user.ID)
		if err != nil {
			return nil, templateError(err)
		}
		return &PublishedRevisionOutput{Body: published}, nil
	})
}

func templateError(err error) error {
	switch {
	case errors.Is(err, templatedomain.ErrNotFound):
		return huma.Error404NotFound("Roteiro não encontrado")
	case errors.Is(err, templatedomain.ErrConflict):
		return huma.Error409Conflict("O rascunho foi alterado em outra sessão")
	case errors.Is(err, templatedomain.ErrNothingToPublish):
		return huma.Error422UnprocessableEntity(
			"O rascunho é igual à revisão publicada",
		)
	case errors.Is(err, templatedomain.ErrInvalidBlueprint):
		return huma.Error422UnprocessableEntity("O roteiro é inválido")
	default:
		slog.Error("template operation failed", "error", err)
		return huma.Error503ServiceUnavailable("Roteiros temporariamente indisponíveis")
	}
}
