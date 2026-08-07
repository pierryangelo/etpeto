package api

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

type Dependencies struct {
	Readiness ReadinessChecker
	Renderer  PreviewRenderer
	Auth      AuthService
	Documents DocumentService
	Templates TemplateService
}

// qualifiedSchemaNamer prefixes a schema with the package that declares it.
// Without this, two domains that both have a Summary or a Section collide in
// the generated document, and the collision only surfaces at runtime.
func qualifiedSchemaNamer(schemaType reflect.Type, hint string) string {
	name := huma.DefaultSchemaNamer(schemaType, hint)
	for schemaType.Kind() == reflect.Ptr || schemaType.Kind() == reflect.Slice {
		schemaType = schemaType.Elem()
	}
	path := schemaType.PkgPath()
	if path == "" {
		return name
	}
	segments := strings.Split(path, "/")
	pkg := segments[len(segments)-1]
	if pkg == "" || strings.HasPrefix(name, capitalise(pkg)) {
		return name
	}
	return capitalise(pkg) + name
}

func capitalise(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func NewRouter(dependencies Dependencies) http.Handler {
	mux := http.NewServeMux()
	config := huma.DefaultConfig("Etpeto API", "0.1.0")
	config.Components.Schemas = huma.NewMapRegistry(
		"#/components/schemas/",
		qualifiedSchemaNamer,
	)
	api := humago.New(mux, config)
	protected := guard{authentication: dependencies.Auth}
	registerHealth(api, dependencies.Readiness)
	registerAuth(api, dependencies.Auth)
	registerDocuments(
		api,
		dependencies.Documents,
		dependencies.Renderer,
		dependencies.Templates,
		protected,
	)
	registerTemplates(api, dependencies.Templates, protected)
	return mux
}
