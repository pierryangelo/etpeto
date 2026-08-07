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

	"github.com/etpeto/etpeto/internal/renderer"
)

const defaultAddress = ":8081"

func main() {
	address := os.Getenv("ETPETO_RENDERER_ADDR")
	if address == "" {
		address = defaultAddress
	}
	maxConcurrent := 2
	if configured := os.Getenv("ETPETO_RENDERER_MAX_CONCURRENT"); configured != "" {
		value, err := strconv.Atoi(configured)
		if err != nil || value < 1 {
			slog.Error("invalid renderer concurrency configuration")
			os.Exit(1)
		}
		maxConcurrent = value
	}

	runner, err := renderer.NewCLIRunner(os.Getenv("ETPETO_RENDERER_TEMP_DIR"), maxConcurrent, 15*time.Second)
	if err != nil {
		slog.Error("invalid renderer configuration", "error", err)
		os.Exit(1)
	}
	server := &http.Server{
		Addr:              address,
		Handler:           renderer.NewHandler(runner),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("Etpeto renderer listening", "address", address)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("renderer shutdown failed", "error", err)
			os.Exit(1)
		}
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("renderer stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}
}
