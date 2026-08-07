package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/etpeto/etpeto/internal/api"
	"github.com/etpeto/etpeto/internal/platform/database"
	"github.com/etpeto/etpeto/internal/renderclient"
)

const defaultAddress = ":8080"

type config struct {
	Address       string
	DatabaseURL   string
	RendererURL   string
	RendererToken string
}

func loadConfig(getenv func(string) string) (config, error) {
	configuration := config{
		Address:       getenv("ETPETO_API_ADDR"),
		DatabaseURL:   getenv("ETPETO_DATABASE_URL"),
		RendererURL:   getenv("ETPETO_RENDERER_URL"),
		RendererToken: getenv("ETPETO_RENDERER_TOKEN"),
	}
	if configuration.Address == "" {
		configuration.Address = defaultAddress
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

	server := &http.Server{
		Addr: configuration.Address,
		Handler: api.NewRouter(api.Dependencies{
			Readiness: database.Checker{Pinger: pool},
			Renderer: renderclient.Client{
				BaseURL: configuration.RendererURL,
				Token:   configuration.RendererToken,
				HTTP:    &http.Client{Timeout: 20 * time.Second},
			},
		}),
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
