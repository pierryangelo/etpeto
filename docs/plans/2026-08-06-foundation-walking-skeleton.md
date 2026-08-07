# Etpeto Foundation Walking Skeleton Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Entregar um Etpeto executável em Docker com interface React visível, API Go documentada, PostgreSQL, renderer Typst interno e autenticação local mínima.

**Architecture:** O corte mantém um monólito modular em Go para API e um segundo binário Go para o renderer, ambos no mesmo módulo. A web React acessa a API pelo mesmo domínio; Caddy serve os arquivos e encaminha `/api/*`. PostgreSQL persiste identidade e sessões, enquanto o renderer aceita somente um contrato JSON fechado.

**Tech Stack:** Go 1.25, Huma v2, `net/http`, pgx v5, Goose, PostgreSQL 17, React, Vite, Bun, TypeScript, Base UI, Lexical, TanStack Query, Zod, Vitest, Testing Library, Docker Compose, Caddy 2 e Typst.

## Global Constraints

- Nome de produto na interface: `Etpeto`; nomes técnicos: `etpeto`.
- Licença: `AGPL-3.0-only`; contribuições futuras seguem DCO 1.1.
- O documento é JSON próprio; Lexical edita somente elementos de texto rico.
- Apenas o gateway publica porta; API, PostgreSQL e renderer ficam internos no Compose.
- O renderer não aceita código Typst, caminho de arquivo ou URL fornecidos pelo usuário.
- O navegador recebe apenas sessão opaca em cookie; tokens de identidade não chegam ao React.
- Desenvolvimento segue teste falhando, implementação mínima, teste verde e refatoração.
- Não adicionar ORM, Redis, broker, Tailwind, Redux ou Zustand.
- Comandos canônicos devem funcionar via `make` localmente e em CI.
- A interface inicial segue a oficina de três painéis recolhíveis e a linguagem Cloudflare arejada aprovada na spec.

---

## File Map

```text
AGENTS.md                         regras curtas para agentes
RTK.md                            convenção de comandos de terminal
Makefile                          comandos canônicos
.env.example                      configuração local sem segredos
compose.yaml                      topologia executável
deploy/Caddyfile                  arquivos estáticos e proxy same-origin
deploy/api.Dockerfile             imagem da API
deploy/renderer.Dockerfile        imagem isolada do renderer
deploy/web.Dockerfile             build e gateway web
cmd/api/main.go                   processo da API
cmd/renderer/main.go              processo interno do renderer
internal/api/router.go            composição HTTP/Huma da API
internal/api/health.go            liveness, readiness e service info
internal/platform/database/       conexão pgx e readiness
internal/auth/                    senha, sessão e casos de uso locais
internal/rendercontract/          JSON compartilhado por API e renderer
internal/renderer/                normalização, Typst e execução limitada
db/migrations/                    schema SQL controlado por Goose
web/src/app/                      composição, rotas e providers React
web/src/features/editor/          oficina visual e estado dos painéis
web/src/features/auth/            login local mínimo
web/src/shared/                   tokens, componentes e cliente HTTP
.github/workflows/ci.yml          gates de pull request
docs/runbooks/development.md      execução e diagnóstico local
```

### Task 1: Repository contract and canonical commands

**Files:**
- Create: `AGENTS.md`
- Create: `RTK.md`
- Create: `Makefile`
- Create: `.env.example`
- Modify: `.gitignore`
- Create: `LICENSE`

**Interfaces:**
- Consumes: product decisions from `docs/specs/2026-08-06-etpeto-design.md`.
- Produces: `make help`, `make fmt`, `make test`, `make build`, `make check`, `make up`, and `make down` for every later task.

- [ ] **Step 1: Write the repository contract check**

Create `scripts/check-repository.sh`:

```sh
#!/bin/sh
set -eu

for path in AGENTS.md RTK.md Makefile LICENSE docs/specs/2026-08-06-etpeto-design.md; do
  test -s "$path" || { echo "missing required file: $path" >&2; exit 1; }
done

grep -q '^help:' Makefile
grep -q 'AGPL-3.0' LICENSE
```

- [ ] **Step 2: Run the check and verify it fails**

Run: `sh scripts/check-repository.sh`  
Expected: FAIL on the first missing root file.

- [ ] **Step 3: Add the minimal repository files**

`AGENTS.md` must point agents to the approved spec and plan, require TDD, forbid secrets and require updating docs when contracts change. `RTK.md` must require `rg` for search and `rtk git` for concise Git output when available. The Makefile must expose:

```make
.PHONY: help fmt test build check up down
help:
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "%-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
fmt: ## Format all source
	gofmt -w $$(find cmd internal -name '*.go' 2>/dev/null) 2>/dev/null || true
	cd web && bun run format
test: ## Run unit tests
	go test ./...
	cd web && bun run test
build: ## Build API, renderer and web
	go build ./cmd/api ./cmd/renderer
	cd web && bun run build
check: fmt test build ## Run required local gates
up: ## Start the reference stack
	docker compose up --build
down: ## Stop the reference stack
	docker compose down
```

Copy the unmodified GNU AGPL v3 license text into `LICENSE`; use `AGPL-3.0-only` in metadata and documentation.

- [ ] **Step 4: Run the repository check**

Run: `sh scripts/check-repository.sh && make help`  
Expected: PASS and a list of canonical commands.

- [ ] **Step 5: Commit**

```bash
git add AGENTS.md RTK.md Makefile .env.example .gitignore LICENSE scripts/check-repository.sh
git commit -m "chore: establish repository contract"
```

### Task 2: Go API with observable health contract

**Files:**
- Create: `go.mod`
- Create: `go.sum`
- Create: `cmd/api/main.go`
- Create: `internal/api/router.go`
- Create: `internal/api/health.go`
- Test: `internal/api/health_test.go`

**Interfaces:**
- Consumes: environment variable `ETPETO_API_ADDR`, default `:8080`.
- Produces: `api.NewRouter(readiness ReadinessChecker) http.Handler`; `GET /livez`, `GET /readyz`, `GET /api/v1/service-info`, and OpenAPI at `/openapi.json`.

- [ ] **Step 1: Write failing health tests**

```go
func TestLiveness(t *testing.T) {
	h := NewRouter(readyStub{err: nil})
	r := httptest.NewRequest(http.MethodGet, "/livez", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestReadinessFailsWhenDependencyIsUnavailable(t *testing.T) {
	h := NewRouter(readyStub{err: errors.New("database unavailable")})
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}
```

- [ ] **Step 2: Run tests and verify the red state**

Run: `go test ./internal/api -run 'Test(Liveness|Readiness)'`  
Expected: FAIL because `NewRouter` and `ReadinessChecker` do not exist.

- [ ] **Step 3: Implement the Huma router**

Define the exact interface:

```go
type ReadinessChecker interface {
	Ready(context.Context) error
}

func NewRouter(readiness ReadinessChecker) http.Handler
```

Use `http.NewServeMux()` with `humago.New(mux, huma.DefaultConfig("Etpeto API", "0.1.0"))`. Return JSON bodies shaped as:

```go
type HealthBody struct {
	Status string `json:"status" example:"ok"`
}

type ServiceInfoBody struct {
	Name    string `json:"name" example:"Etpeto"`
	Version string `json:"version" example:"0.1.0-dev"`
}
```

Readiness errors must return `503` without exposing the dependency error text. `main.go` must configure timeouts and perform graceful shutdown on `SIGINT` or `SIGTERM`.

- [ ] **Step 4: Run API tests and race detector**

Run: `go test -race ./internal/api`  
Expected: PASS.

- [ ] **Step 5: Verify generated OpenAPI**

Add a test that requests `/openapi.json` and asserts the document contains `/api/v1/service-info`, `/livez` and `/readyz`.

Run: `go test ./internal/api -run TestOpenAPI`  
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum cmd/api internal/api
git commit -m "feat(api): add health and service contract"
```

### Task 3: PostgreSQL foundation and readiness

**Files:**
- Create: `internal/platform/database/pool.go`
- Create: `internal/platform/database/readiness.go`
- Test: `internal/platform/database/readiness_test.go`
- Create: `db/migrations/00001_foundation.sql`
- Create: `db/queries/users.sql`
- Create: `sqlc.yaml`
- Create: `goose.yaml`

**Interfaces:**
- Consumes: `ETPETO_DATABASE_URL` and `context.Context`.
- Produces: `database.Open(ctx, url) (*pgxpool.Pool, error)` and `database.Checker{Pinger Pinger}.Ready(ctx) error` satisfying `api.ReadinessChecker`; `*pgxpool.Pool` satisfies `Pinger`.

- [ ] **Step 1: Write the failing readiness unit test**

Define a private `pinger` interface with `Ping(context.Context) error`, inject a stub and assert that `Checker.Ready` returns its error unchanged.

Run: `go test ./internal/platform/database`  
Expected: FAIL because `Checker` does not exist.

- [ ] **Step 2: Implement connection and readiness**

`Open` must reject an empty URL, use `pgxpool.New`, ping during startup and close the pool on startup failure. `Checker.Ready` delegates to `Ping` with the request context.

- [ ] **Step 3: Add the first reversible migration**

The Goose migration must create UUID-capable tables `organizations`, `users`, `identities`, `memberships` and `sessions` with:

- `created_at timestamptz not null default now()`;
- case-insensitive normalized local email uniqueness per organization;
- external identity uniqueness on `(issuer, subject)`;
- membership uniqueness on `(organization_id, user_id)`;
- session `token_hash bytea` unique, expiry, revocation timestamp and user foreign key.

The down migration drops these tables in reverse dependency order.

- [ ] **Step 4: Generate sqlc and run integration test**

Add a Testcontainers test which starts PostgreSQL 17, runs Goose up, inserts one organization and user through generated queries, then proves duplicate normalized email is rejected.

Run: `go test -tags=integration ./internal/platform/database`  
Expected: PASS when Docker is available.

- [ ] **Step 5: Wire database readiness into `cmd/api`**

Startup must fail with a safe error when `ETPETO_DATABASE_URL` is absent or unavailable. Shutdown closes the HTTP server before the pool.

Run: `go test -race ./...`  
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/platform/database db sqlc.yaml goose.yaml cmd/api go.mod go.sum
git commit -m "feat(database): add PostgreSQL foundation"
```

### Task 4: Closed renderer contract and Typst source builder

**Files:**
- Create: `internal/rendercontract/document.go`
- Create: `internal/rendercontract/document_test.go`
- Create: `internal/renderer/source.go`
- Create: `internal/renderer/source_test.go`
- Create: `internal/renderer/runner.go`
- Create: `internal/renderer/service.go`
- Test: `internal/renderer/service_test.go`
- Create: `cmd/renderer/main.go`

**Interfaces:**
- Consumes: `rendercontract.Request` JSON and environment `ETPETO_RENDERER_ADDR`, default `:8081`.
- Produces: `renderer.BuildSource(req rendercontract.Request) (string, error)`, `renderer.Runner.Render(ctx, source) ([]byte, error)`, `POST /internal/render`, and `GET /livez`.

- [ ] **Step 1: Define the contract through failing tests**

```go
func TestBuildSourceEscapesTypstSyntaxAndKeepsURLsAsText(t *testing.T) {
	req := rendercontract.Request{
		SchemaVersion: 1,
		Title:         `#import "evil.typ"`,
		Sections: []rendercontract.Section{{
			Title:      "Referências [externas]",
			Paragraphs: []string{`Consulte https://example.test e C:\\temp`},
		}},
	}
	source, err := BuildSource(req)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(source, `#import "evil.typ"`) {
		t.Fatal("user text was emitted as executable Typst")
	}
	if !strings.Contains(source, "https://example.test") {
		t.Fatal("ordinary URL text was lost")
	}
}
```

The initial request fields are exactly:

```go
type Request struct {
	SchemaVersion      int       `json:"schemaVersion"`
	TemplateRevisionID string    `json:"templateRevisionId"`
	Title              string    `json:"title"`
	Sections           []Section `json:"sections"`
}

type Section struct {
	Title      string   `json:"title"`
	Paragraphs []string `json:"paragraphs"`
}
```

- [ ] **Step 2: Run the contract tests and verify failure**

Run: `go test ./internal/rendercontract`  
Expected: FAIL because the request types and validation are absent.

- [ ] **Step 3: Implement bounded validation and escaping**

Accept schema version `1`, 1–100 sections, titles up to 300 Unicode code points and paragraphs up to 20,000 code points. Text is data and must be escaped by `BuildSource`; ordinary `https://` text is allowed and rendered literally, while no field is interpreted as a fetch instruction. Assert that generated source neutralizes `#`, `[`, `]`, `\\` and quotes according to Typst string/content rules.

- [ ] **Step 4: Test the renderer service with a fake runner**

```go
type Runner interface {
	Render(context.Context, string) ([]byte, error)
}
```

`POST /internal/render` accepts at most 2 MiB, validates JSON, calls the runner and returns `application/pdf` plus `ETag` containing the SHA-256 hash. A runner error returns Problem Details `503` without the Typst stderr body.

Run: `go test -race ./internal/renderer`  
Expected: PASS.

- [ ] **Step 5: Implement the bounded Typst CLI runner**

Implement this as `CLIRunner`. Use `exec.CommandContext(ctx, "typst", "compile", "--format", "pdf", inputPath, outputPath)`. Create files only inside an injected temporary directory, set a 15-second parent timeout, cap concurrent renders with a buffered semaphore and read the resulting PDF only after a zero exit status.

- [ ] **Step 6: Commit**

```bash
git add internal/rendercontract internal/renderer cmd/renderer
git commit -m "feat(renderer): add closed Typst render contract"
```

### Task 5: API bridge to the internal renderer

**Files:**
- Create: `internal/renderclient/client.go`
- Test: `internal/renderclient/client_test.go`
- Create: `internal/api/preview.go`
- Test: `internal/api/preview_test.go`
- Modify: `internal/api/router.go`
- Modify: `cmd/api/main.go`

**Interfaces:**
- Consumes: `ETPETO_RENDERER_URL`, `rendercontract.Request` and renderer `POST /internal/render`.
- Produces: `renderclient.Client.Render(ctx, request) (PDF, error)` and public `POST /api/v1/previews` returning `application/pdf` with `ETag`.

- [ ] **Step 1: Write a failing renderer client test**

Use `httptest.Server` to assert the client sends `application/json`, the closed render contract and `X-Etpeto-Renderer-Token`; return a PDF fixture and assert bytes plus ETag are preserved.

Run: `go test ./internal/renderclient`  
Expected: FAIL because `Client` does not exist.

- [ ] **Step 2: Implement the bounded client**

Define:

```go
type PDF struct {
	Bytes []byte
	ETag  string
}

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func (c Client) Render(ctx context.Context, request rendercontract.Request) (PDF, error)
```

Reject non-2xx responses, non-PDF content types and bodies above 20 MiB. The HTTP client timeout is 20 seconds. Do not include the renderer response body in returned errors.

- [ ] **Step 3: Write the failing public preview route test**

Inject a fake with `Render(context.Context, rendercontract.Request) (renderclient.PDF, error)`. Assert valid requests return PDF and ETag; validation errors return `422`; renderer errors return Problem Details `503`.

- [ ] **Step 4: Register the route and wire configuration**

Extend `NewRouter` with a dependencies struct:

```go
type Dependencies struct {
	Readiness ReadinessChecker
	Renderer  PreviewRenderer
}

func NewRouter(deps Dependencies) http.Handler
```

`cmd/api` reads renderer URL and token from environment and fails startup when either is empty.

- [ ] **Step 5: Run bridge tests**

Run: `go test -race ./internal/renderclient ./internal/api`  
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/renderclient internal/api cmd/api
git commit -m "feat(api): bridge preview rendering"
```

### Task 6: React application shell and approved editor workspace

**Files:**
- Create: `web/package.json`
- Create: `web/bun.lock`
- Create: `web/tsconfig.json`
- Create: `web/vite.config.ts`
- Create: `web/vitest.setup.ts`
- Create: `web/index.html`
- Create: `web/src/main.tsx`
- Create: `web/src/app/App.tsx`
- Create: `web/src/app/App.test.tsx`
- Create: `web/src/features/editor/EditorWorkspace.tsx`
- Create: `web/src/features/editor/EditorWorkspace.module.css`
- Test: `web/src/features/editor/EditorWorkspace.test.tsx`
- Create: `web/src/shared/styles/tokens.css`
- Create: `web/src/shared/styles/global.css`

**Interfaces:**
- Consumes: no backend data yet; an in-file typed fixture represents “Aquisição de notebooks”.
- Produces: accessible Etpeto shell with `Roteiro`, `Editor` and `Prévia oficial`, independently collapsible side panels and mobile mode navigation.

- [ ] **Step 1: Bootstrap only the test runner configuration**

Install React, Vite, TypeScript, Vitest, Testing Library, jsdom, Prettier, `@base-ui/react`, TanStack Query, Zod, React Hook Form and Lexical packages with Bun. Configure Vitest for jsdom and import `@testing-library/jest-dom/vitest` in `vitest.setup.ts`. Package scripts must expose `test` as `vitest run`, `typecheck` as `tsc --noEmit`, `format` as `prettier --write .` and `build` as `tsc -b && vite build`.

- [ ] **Step 2: Write failing workspace behavior tests**

```tsx
it('recolhe o roteiro sem esconder o editor', async () => {
  const user = userEvent.setup();
  render(<EditorWorkspace />);
  await user.click(screen.getByRole('button', { name: 'Recolher roteiro' }));
  expect(screen.queryByRole('navigation', { name: 'Roteiro do documento' })).not.toBeInTheDocument();
  expect(screen.getByRole('main')).toBeVisible();
  expect(screen.getByRole('button', { name: 'Expandir roteiro' })).toBeVisible();
});

it('oferece modos exclusivos no viewport móvel', () => {
  render(<EditorWorkspace compact />);
  expect(screen.getByRole('tab', { name: 'Editar' })).toHaveAttribute('aria-selected', 'true');
  expect(screen.getByRole('tab', { name: 'Roteiro' })).toBeVisible();
  expect(screen.getByRole('tab', { name: 'Prévia' })).toBeVisible();
});
```

- [ ] **Step 3: Run the test and verify the red state**

Run: `cd web && bun run test src/features/editor/EditorWorkspace.test.tsx`  
Expected: FAIL because `EditorWorkspace` does not exist.

- [ ] **Step 4: Build the minimum accessible workspace**

Use semantic landmarks and Base UI primitives where interaction requires them. Persist desktop collapse state under `etpeto:editor-layout:v1`. Do not persist document content in this task. Use CSS grid for desktop and tabs for compact mode.

Tokens must include surface, text, border, accent orange, success, danger, focus ring, spacing, radius and shadow values. The root must use `isolation: isolate`; `body` uses `position: relative` for Base UI portal behavior.

- [ ] **Step 5: Implement the approved visual content**

Render real labels from the approved mock:

- document `Aquisição de notebooks`;
- sections `Necessidade`, `Requisitos`, `Pesquisa de preços`, `Mapa de riscos`, `Conclusão`;
- save state `Salvo agora`;
- actions `Finalizar` and `Exportar PDF`;
- PDF panel explicitly labeled `Prévia oficial`.

The visual must be Cloudflare-airy in spacing and hierarchy without copying its logo or branded palette.

- [ ] **Step 6: Run frontend gates**

Run: `cd web && bun run test && bun run typecheck && bun run build`  
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add web
git commit -m "feat(web): add Etpeto editor workspace"
```

### Task 7: Lexical rich-text element and local recovery

**Files:**
- Create: `web/src/features/editor/RichTextElement.tsx`
- Create: `web/src/features/editor/RichTextElement.module.css`
- Test: `web/src/features/editor/RichTextElement.test.tsx`
- Create: `web/src/features/editor/recovery.ts`
- Test: `web/src/features/editor/recovery.test.ts`

**Interfaces:**
- Consumes: `documentId`, `elementId`, optional serialized Lexical state and `onChange(state: SerializedEditorState): void`.
- Produces: `RichTextElement` and `createRecoveryStore(storage: Storage)` with `load`, `save`, and `clearPersistedRevision` methods.

- [ ] **Step 1: Write failing recovery tests**

Test that keys use `etpeto:recovery:v1:<documentId>`, stale saves cannot clear a newer local revision and invalid JSON is ignored without throwing.

Run: `cd web && bun run test src/features/editor/recovery.test.ts`  
Expected: FAIL because `createRecoveryStore` is absent.

- [ ] **Step 2: Implement the recovery store**

Persist an envelope containing `schemaVersion`, `documentId`, `serverRevision`, `savedAt` and `content`. `clearPersistedRevision` may delete only when the acknowledged revision is greater than or equal to the stored revision.

- [ ] **Step 3: Write the failing Lexical round-trip test**

Mount the editor with a paragraph containing “Descreva as condições indispensáveis”, change the text, and assert `onChange` receives a Lexical JSON state whose root contains the new text.

- [ ] **Step 4: Implement the minimal Lexical editor**

Enable paragraphs, headings, lists, bold, italic, underline and links. Do not enable arbitrary HTML import, tables, custom domain blocks or collaboration plugins. Toolbar buttons require accessible names and pressed state.

- [ ] **Step 5: Run frontend tests**

Run: `cd web && bun run test && bun run typecheck`  
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add web/src/features/editor
git commit -m "feat(editor): add rich text and local recovery"
```

### Task 8: Minimal local authentication slice

**Files:**
- Create: `internal/auth/password.go`
- Test: `internal/auth/password_test.go`
- Create: `internal/auth/session.go`
- Test: `internal/auth/session_test.go`
- Create: `internal/auth/service.go`
- Test: `internal/auth/service_test.go`
- Create: `internal/api/auth.go`
- Test: `internal/api/auth_test.go`
- Create: `web/src/features/auth/LoginPage.tsx`
- Test: `web/src/features/auth/LoginPage.test.tsx`

**Interfaces:**
- Consumes: normalized email, password and PostgreSQL repositories generated by sqlc.
- Produces: `POST /api/v1/auth/local/login`, `POST /api/v1/auth/logout`, `GET /api/v1/auth/session`; cookie `__Host-etpeto_session` with `Secure`, `HttpOnly`, `SameSite=Lax` and `Path=/`.

- [ ] **Step 1: Write failing password tests**

Test Argon2id hash/verify success, wrong-password failure, malformed-hash failure and a maximum accepted password byte length of 1024.

- [ ] **Step 2: Implement password hashing**

Use `golang.org/x/crypto/argon2`, a 16-byte random salt, 32-byte key and parameters stored in the encoded hash. Production defaults must meet the documented project floor; tests inject lower parameters for speed.

- [ ] **Step 3: Write failing session tests**

Test 32 random bytes encoded with base64url, SHA-256 token hashing, absolute expiration, revocation and constant-time hash comparison.

- [ ] **Step 4: Implement service and repositories**

Normalize email with trim plus lowercase, return one generic credential error, and create the session only after successful password verification. Persist only the token hash. Never log email plus authentication result at info level.

- [ ] **Step 5: Add API contract tests**

Assert successful login sends HttpOnly, SameSite Lax, Path `/` cookie; invalid login returns identical `401` body for missing user and wrong password; logout revokes the current session and expires the cookie.

- [ ] **Step 6: Add the login page test and implementation**

The page must have labeled email/password fields, submit on Enter, announce server errors, disable duplicate submission and navigate to `/documentos` after session confirmation.

- [ ] **Step 7: Run auth gates**

Run: `go test -race ./internal/auth ./internal/api && cd web && bun run test src/features/auth`  
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/auth internal/api db/queries web/src/features/auth web/src/app
git commit -m "feat(auth): add local session login"
```

### Task 9: Docker reference stack and same-origin gateway

**Files:**
- Create: `compose.yaml`
- Create: `deploy/Caddyfile`
- Create: `deploy/api.Dockerfile`
- Create: `deploy/renderer.Dockerfile`
- Create: `deploy/web.Dockerfile`
- Create: `scripts/smoke-compose.sh`
- Test: `scripts/smoke-compose.sh`

**Interfaces:**
- Consumes: `.env` based on `.env.example`.
- Produces: public `http://localhost:8080`, same-origin `/api/*`, internal `api:8080`, `renderer:8081`, `postgres:5432` and one-shot `migrate`.

- [ ] **Step 1: Write the failing smoke script**

```sh
#!/bin/sh
set -eu
base_url="${ETPETO_BASE_URL:-http://localhost:8080}"
curl --fail --silent "$base_url/api/v1/service-info" | grep -q '"name":"Etpeto"'
curl --fail --silent "$base_url/" | grep -q '<title>Etpeto</title>'
```

Run: `sh scripts/smoke-compose.sh`  
Expected: FAIL because the stack is not running.

- [ ] **Step 2: Create non-root multi-stage images**

The API and renderer final images contain only the binary, CA certificates and an unprivileged user. The renderer additionally contains a pinned Typst binary and writable `/tmp/etpeto` tmpfs only. The web image builds with Bun, then copies `dist` into Caddy.

- [ ] **Step 3: Define Compose ordering and networks**

Create `edge`, `data` and `render` networks. `migrate` depends on healthy PostgreSQL; API depends on successful migration; web depends on API; renderer is attached only to the `render` network and has no published port. Add healthchecks without leaking credentials.

- [ ] **Step 4: Run the stack and smoke contract**

Run: `docker compose up --build -d && sh scripts/smoke-compose.sh`  
Expected: PASS.

- [ ] **Step 5: Inspect isolation**

Run: `docker compose ps` and `docker compose config`  
Expected: only `web` publishes `8080`; renderer has no database environment variable.

- [ ] **Step 6: Commit**

```bash
git add compose.yaml deploy scripts/smoke-compose.sh .env.example
git commit -m "feat(deploy): add isolated Docker stack"
```

### Task 10: CI, public contribution baseline and developer runbook

**Files:**
- Create: `.github/workflows/ci.yml`
- Create: `.github/dependabot.yml`
- Create: `.github/CODEOWNERS`
- Create: `.github/pull_request_template.md`
- Create: `README.md`
- Create: `CONTRIBUTING.md`
- Create: `CODE_OF_CONDUCT.md`
- Create: `SECURITY.md`
- Create: `GOVERNANCE.md`
- Create: `SUPPORT.md`
- Create: `docs/runbooks/development.md`
- Create: `docs/architecture/foundation.md`

**Interfaces:**
- Consumes: canonical commands and working Compose stack from prior tasks.
- Produces: reproducible onboarding and a PR gate invoking `make check` plus integration smoke.

- [ ] **Step 1: Add a documentation contract check**

Extend `scripts/check-repository.sh` to require every public-community file above and verify README contains quick-start, architecture, security reporting and license links.

Run: `sh scripts/check-repository.sh`  
Expected: FAIL listing the absent public files.

- [ ] **Step 2: Write concise public documentation**

README must explain the ETP problem, show the approved editor direction, run `cp .env.example .env && make up`, identify `http://localhost:8080`, and explicitly label the repository as pre-`v0.1`. SECURITY must direct private vulnerability reports to GitHub private vulnerability reporting, not public issues. CONTRIBUTING must require DCO sign-off and tests.

- [ ] **Step 3: Add least-privilege CI**

CI triggers on pull requests and pushes to `main`, uses `permissions: contents: read`, pins actions by full commit SHA, caches Go and Bun dependencies, runs unit gates first and Compose smoke second. Fork PRs receive no secrets.

- [ ] **Step 4: Run all local evidence**

Run: `make check`  
Expected: PASS.

Run: `docker compose up --build -d && sh scripts/smoke-compose.sh`  
Expected: PASS.

Run: `docker compose down -v`  
Expected: stack stops and test-only volumes are removed.

- [ ] **Step 5: Commit**

```bash
git add .github README.md CONTRIBUTING.md CODE_OF_CONDUCT.md SECURITY.md GOVERNANCE.md SUPPORT.md docs scripts/check-repository.sh
git commit -m "docs: prepare Etpeto for public contribution"
```

## Completion Evidence

Before claiming this plan complete, collect and report:

```bash
git status --short
make check
go test -race ./...
cd web && bun run test --coverage && bun run typecheck && bun run build
docker compose up --build -d
sh scripts/smoke-compose.sh
docker compose ps
docker compose down
```

The handoff must include the exact commit range, commands executed, failures or skips, and the URL of the visible Etpeto shell. A green unit suite without the Compose smoke test is not completion of this plan.
