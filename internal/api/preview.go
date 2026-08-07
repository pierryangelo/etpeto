package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/etpeto/etpeto/internal/renderclient"
	"github.com/etpeto/etpeto/internal/rendercontract"
)

type PreviewRenderer interface {
	Render(context.Context, rendercontract.Request) (renderclient.PDF, error)
}

type PreviewInput struct {
	Body rendercontract.Request
}

type PreviewOutput struct {
	ContentType string `header:"Content-Type"`
	ETag        string `header:"ETag"`
	Body        []byte
}

func registerPreview(api huma.API, renderer PreviewRenderer) {
	huma.Register(api, huma.Operation{
		OperationID:  "create-preview",
		Method:       http.MethodPost,
		Path:         "/api/v1/previews",
		Summary:      "Render an official PDF preview",
		Tags:         []string{"Documents"},
		MaxBodyBytes: 2 << 20,
		Responses: map[string]*huma.Response{
			"200": {
				Description: "Rendered PDF preview",
				Content: map[string]*huma.MediaType{
					"application/pdf": {},
				},
			},
		},
	}, func(ctx context.Context, input *PreviewInput) (*PreviewOutput, error) {
		if err := input.Body.Validate(); err != nil {
			return nil, huma.Error422UnprocessableEntity("Solicitação de prévia inválida")
		}
		if renderer == nil {
			return nil, huma.Error503ServiceUnavailable("Renderização temporariamente indisponível")
		}
		pdf, err := renderer.Render(ctx, input.Body)
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
