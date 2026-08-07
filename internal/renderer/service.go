package renderer

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/etpeto/etpeto/internal/rendercontract"
)

const maxRequestBytes = 2 << 20

type problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Code   string `json:"code"`
}

func NewHandler(runner Runner, token string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /internal/render", func(response http.ResponseWriter, request *http.Request) {
		providedToken := request.Header.Get("X-Etpeto-Renderer-Token")
		if token == "" || subtle.ConstantTimeCompare([]byte(providedToken), []byte(token)) != 1 {
			writeProblem(response, http.StatusUnauthorized, "renderer_authentication_required", "Autenticação interna inválida")
			return
		}
		render(response, request, runner)
	})
	return mux
}

func render(response http.ResponseWriter, request *http.Request, runner Runner) {
	if request.ContentLength > maxRequestBytes {
		writeProblem(response, http.StatusRequestEntityTooLarge, "render_payload_too_large", "Payload de renderização muito grande")
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, maxRequestBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	var renderRequest rendercontract.Request
	if err := decoder.Decode(&renderRequest); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeProblem(response, http.StatusRequestEntityTooLarge, "render_payload_too_large", "Payload de renderização muito grande")
			return
		}
		writeProblem(response, http.StatusUnprocessableEntity, "invalid_render_request", "Solicitação de renderização inválida")
		return
	}
	if err := ensureJSONEnd(decoder); err != nil {
		writeProblem(response, http.StatusUnprocessableEntity, "invalid_render_request", "Solicitação de renderização inválida")
		return
	}

	source, err := BuildSource(renderRequest)
	if err != nil {
		writeProblem(response, http.StatusUnprocessableEntity, "invalid_render_request", "Solicitação de renderização inválida")
		return
	}
	if runner == nil {
		writeProblem(response, http.StatusServiceUnavailable, "renderer_unavailable", "Renderização temporariamente indisponível")
		return
	}
	pdf, err := runner.Render(request.Context(), source)
	if err != nil {
		writeProblem(response, http.StatusServiceUnavailable, "renderer_unavailable", "Renderização temporariamente indisponível")
		return
	}

	hash := sha256.Sum256(pdf)
	response.Header().Set("Content-Type", "application/pdf")
	response.Header().Set("ETag", `"`+hex.EncodeToString(hash[:])+`"`)
	response.Header().Set("Content-Length", strconv.Itoa(len(pdf)))
	response.WriteHeader(http.StatusOK)
	_, _ = response.Write(pdf)
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("multiple JSON values")
	}
	return err
}

func writeProblem(response http.ResponseWriter, status int, code string, title string) {
	response.Header().Set("Content-Type", "application/problem+json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(problem{
		Type:   "about:blank",
		Title:  title,
		Status: status,
		Code:   code,
	})
}
