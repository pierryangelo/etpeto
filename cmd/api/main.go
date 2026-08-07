package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/etpeto/etpeto/internal/api"
	authdomain "github.com/etpeto/etpeto/internal/auth"
	documentdomain "github.com/etpeto/etpeto/internal/document"
	"github.com/etpeto/etpeto/internal/platform/database"
	"github.com/etpeto/etpeto/internal/platform/database/dbgen"
	"github.com/etpeto/etpeto/internal/renderclient"
	templatedomain "github.com/etpeto/etpeto/internal/template"
)

const defaultAddress = ":8080"

type config struct {
	Address          string
	TrustProxy       bool
	ThrottleScale    int
	DatabaseURL      string
	RendererURL      string
	RendererToken    string
	OrganizationSlug string
}

func loadConfig(getenv func(string) string) (config, error) {
	configuration := config{
		Address:          getenv("ETPETO_API_ADDR"),
		DatabaseURL:      getenv("ETPETO_DATABASE_URL"),
		RendererURL:      getenv("ETPETO_RENDERER_URL"),
		RendererToken:    getenv("ETPETO_RENDERER_TOKEN"),
		OrganizationSlug: getenv("ETPETO_ORGANIZATION_SLUG"),
		// The API sits behind the gateway in the reference stack, so the
		// forwarded address is the real client. Exposing the API directly would
		// make that header attacker-controlled, hence a deliberate setting.
		TrustProxy: getenv("ETPETO_TRUST_PROXY") != "false",
		// Raise this only where many people reach the API through one address,
		// such as an installation published to the internet behind NAT.
		ThrottleScale: positiveOrOne(getenv("ETPETO_THROTTLE_SCALE")),
	}
	if configuration.Address == "" {
		configuration.Address = defaultAddress
	}
	if configuration.OrganizationSlug == "" {
		configuration.OrganizationSlug = "default"
	}
	if configuration.DatabaseURL == "" {
		return config{}, errors.New("ETPETO_DATABASE_URL is required")
	}
	if configuration.RendererURL == "" {
		return config{}, errors.New("ETPETO_RENDERER_URL is required")
	}
	if configuration.RendererToken == "" {
		return config{}, errors.New("ETPETO_RENDERER_TOKEN is required")
	}
	return configuration, nil
}

func positiveOrOne(value string) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 1
	}
	return parsed
}

func main() {
	configuration, err := loadConfig(os.Getenv)
	if err != nil {
		slog.Error("invalid API configuration", "error", err)
		os.Exit(1)
	}

	startupCtx, cancelStartup := context.WithTimeout(context.Background(), 10*time.Second)
	pool, err := database.Open(startupCtx, configuration.DatabaseURL)
	cancelStartup()
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	authentication, err := authdomain.NewService(
		authdomain.PostgresRepository{
			Queries:          dbgen.New(pool),
			OrganizationSlug: configuration.OrganizationSlug,
		},
		authdomain.PasswordHasher{},
		authdomain.SessionIssuer{TTL: 12 * time.Hour},
	)
	if err != nil {
		slog.Error("authentication initialization failed", "error", err)
		os.Exit(1)
	}

	router := api.NewRouter(api.Dependencies{
		Readiness: database.Checker{Pinger: pool},
		Renderer: renderclient.Client{
			BaseURL: configuration.RendererURL,
			Token:   configuration.RendererToken,
			HTTP:    &http.Client{Timeout: 20 * time.Second},
		},
		Auth: authentication,
		Documents: documentdomain.Service{
			Repository: documentdomain.PostgresRepository{Pool: pool},
		},
		Templates: templatedomain.Service{
			Repository: templatedomain.PostgresRepository{Pool: pool},
		},
		People: authdomain.Invitations{
			Repository: authdomain.PostgresInvitationRepository{Pool: pool},
			Passwords:  authdomain.PasswordHasher{},
			Tokens:     authdomain.SessionIssuer{TTL: authdomain.InvitationTTL},
		},
	})

	server := &http.Server{
		Addr: configuration.Address,
		Handler: api.Throttle(
			router,
			api.ScaledThrottles(configuration.ThrottleScale),
			configuration.TrustProxy,
		),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("Etpeto API listening", "address", configuration.Address)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("API shutdown failed", "error", err)
			os.Exit(1)
		}
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("API stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}
}
