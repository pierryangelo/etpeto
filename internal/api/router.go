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
}

func NewRouter(dependencies Dependencies) http.Handler {
	mux := http.NewServeMux()
	config := huma.DefaultConfig("Etpeto API", "0.1.0")
	api := humago.New(mux, config)
	registerHealth(api, dependencies.Readiness)
	registerPreview(api, dependencies.Renderer)
	registerAuth(api, dependencies.Auth)
	return mux
}
