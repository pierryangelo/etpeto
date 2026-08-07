package api

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

func NewRouter(readiness ReadinessChecker) http.Handler {
	mux := http.NewServeMux()
	config := huma.DefaultConfig("Etpeto API", "0.1.0")
	api := humago.New(mux, config)
	registerHealth(api, readiness)
	return mux
}
