// Command bootstrap provisions the first local administrator of an Etpeto
// installation. It is idempotent for the same address and refuses to widen
// access on an installation that already has an administrator.
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	authdomain "github.com/etpeto/etpeto/internal/auth"
	"github.com/etpeto/etpeto/internal/platform/database"
)

type environment struct {
	getenv func(string) string
	stdin  io.Reader
	stdout io.Writer
}

func main() {
	code := run(environment{getenv: os.Getenv, stdin: os.Stdin, stdout: os.Stdout})
	os.Exit(code)
}

func run(env environment) int {
	databaseURL := env.getenv("ETPETO_DATABASE_URL")
	if databaseURL == "" {
		slog.Error("ETPETO_DATABASE_URL is required")
		return 1
	}
	password, err := readPassword(env)
	if err != nil {
		slog.Error("could not read the administrator password", "error", err)
		return 1
	}

	request := authdomain.ProvisionRequest{
		OrganizationSlug: valueOrDefault(env.getenv("ETPETO_ORGANIZATION_SLUG"), "default"),
		OrganizationName: env.getenv("ETPETO_ORGANIZATION_NAME"),
		Email:            env.getenv("ETPETO_ADMIN_EMAIL"),
		DisplayName:      env.getenv("ETPETO_ADMIN_DISPLAY_NAME"),
		Password:         password,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		return 1
	}
	defer pool.Close()

	provisioner := authdomain.Provisioner{
		Repository: authdomain.PostgresProvisioningRepository{Pool: pool},
		Passwords:  authdomain.PasswordHasher{},
	}
	outcome, err := provisioner.Provision(ctx, request)
	switch {
	case errors.Is(err, authdomain.ErrInstallationAlreadyInitialized):
		slog.Error(
			"this organization already has an administrator; " +
				"create further users through the application instead of re-running bootstrap",
		)
		return 1
	case errors.Is(err, authdomain.ErrWeakPassword):
		slog.Error("the administrator password is shorter than the accepted floor")
		return 1
	case errors.Is(err, authdomain.ErrInvalidProvisionRequest):
		slog.Error("ETPETO_ADMIN_EMAIL and ETPETO_ADMIN_DISPLAY_NAME are required")
		return 1
	case err != nil:
		slog.Error("provisioning failed", "error", err)
		return 1
	}

	if outcome.Created {
		fmt.Fprintf(env.stdout, "administrator provisioned for organization %s\n", request.OrganizationSlug)
		return 0
	}
	fmt.Fprintf(env.stdout, "administrator already present for organization %s; nothing changed\n", request.OrganizationSlug)
	return 0
}

// readPassword prefers standard input so the credential never appears in the
// process arguments or the shell history. An environment variable remains
// available for secret mounts in automated installs.
func readPassword(env environment) (string, error) {
	if env.stdin != nil {
		reader := bufio.NewReader(env.stdin)
		line, err := reader.ReadString('\n')
		trimmed := strings.TrimRight(line, "\r\n")
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}
		if trimmed != "" {
			return trimmed, nil
		}
	}
	if fromEnvironment := env.getenv("ETPETO_ADMIN_PASSWORD"); fromEnvironment != "" {
		return fromEnvironment, nil
	}
	return "", errors.New("no password on standard input or ETPETO_ADMIN_PASSWORD")
}

func valueOrDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
