package api

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

type Dependencies struct {
	Readiness ReadinessChecker
	Renderer  PreviewRenderer
	Auth      AuthService
	Documents DocumentService
}

func NewRouter(dependencies Dependencies) http.Handler {
	mux := http.NewServeMux()
	config := huma.DefaultConfig("Etpeto API", "0.1.0")
	api := humago.New(mux, config)
	protected := guard{authentication: dependencies.Auth}
	registerHealth(api, dependencies.Readiness)
	registerAuth(api, dependencies.Auth)
	registerDocuments(api, dependencies.Documents, dependencies.Renderer, protected)
	return mux
}
