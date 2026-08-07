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
)

const defaultAddress = ":8080"

type ready struct{}

func (ready) Ready(context.Context) error { return nil }

func main() {
	address := os.Getenv("ETPETO_API_ADDR")
	if address == "" {
		address = defaultAddress
	}

	server := &http.Server{
		Addr:              address,
		Handler:           api.NewRouter(ready{}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("Etpeto API listening", "address", address)
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
