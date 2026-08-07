package renderclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/etpeto/etpeto/internal/rendercontract"
)

const maxPDFBytes = 20 << 20

type PDF struct {
	Bytes []byte
	ETag  string
}

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func (c Client) Render(ctx context.Context, renderRequest rendercontract.Request) (PDF, error) {
	if err := renderRequest.Validate(); err != nil {
		return PDF{}, fmt.Errorf("validate render request: %w", err)
	}
	endpoint, err := rendererEndpoint(c.BaseURL)
	if err != nil {
		return PDF{}, err
	}
	if c.Token == "" {
		return PDF{}, errors.New("renderer token is required")
	}

	body, err := json.Marshal(renderRequest)
	if err != nil {
		return PDF{}, fmt.Errorf("encode render request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return PDF{}, fmt.Errorf("create renderer request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/pdf")
	request.Header.Set("X-Etpeto-Renderer-Token", c.Token)

	response, err := c.httpClient().Do(request)
	if err != nil {
		return PDF{}, fmt.Errorf("call renderer: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return PDF{}, fmt.Errorf("renderer returned HTTP status %d", response.StatusCode)
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/pdf" {
		return PDF{}, errors.New("renderer returned an invalid content type")
	}
	if response.ContentLength > maxPDFBytes {
		return PDF{}, errors.New("renderer PDF exceeds size limit")
	}

	pdf, err := io.ReadAll(io.LimitReader(response.Body, maxPDFBytes+1))
	if err != nil {
		return PDF{}, fmt.Errorf("read renderer PDF: %w", err)
	}
	if len(pdf) > maxPDFBytes {
		return PDF{}, errors.New("renderer PDF exceeds size limit")
	}
	return PDF{Bytes: pdf, ETag: response.Header.Get("ETag")}, nil
}

func (c Client) httpClient() *http.Client {
	client := &http.Client{}
	if c.HTTP != nil {
		*client = *c.HTTP
	}
	if client.Timeout <= 0 || client.Timeout > 20*time.Second {
		client.Timeout = 20 * time.Second
	}
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return client
}

func rendererEndpoint(baseURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", errors.New("renderer URL must be an absolute HTTP URL")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/internal/render"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}
