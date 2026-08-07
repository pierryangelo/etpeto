//go:build integration

package database

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	authdomain "github.com/etpeto/etpeto/internal/auth"
	"github.com/etpeto/etpeto/internal/platform/database/dbgen"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestFoundationMigrationEnforcesNormalizedLocalEmailUniqueness(t *testing.T) {
	ctx := context.Background()
	container, err := postgres.Run(
		ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("etpeto_test"),
		postgres.WithUsername("etpeto"),
		postgres.WithPassword("etpeto_test_password"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start PostgreSQL: %v", err)
	}
	testcontainers.CleanupContainer(t, container)

	databaseURL, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get PostgreSQL connection string: %v", err)
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		db,
		os.DirFS("../../../db/migrations"),
	)
	if err != nil {
		t.Fatalf("create migration provider: %v", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	pool, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open pgx pool: %v", err)
	}
	t.Cleanup(pool.Close)
	queries := dbgen.New(pool)

	organization, err := queries.CreateOrganization(ctx, dbgen.CreateOrganizationParams{
		Slug: "orgao-teste",
		Name: "Órgão de Teste",
	})
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	firstUser, err := queries.CreateUser(ctx, "Primeira Pessoa")
	if err != nil {
		t.Fatalf("create first user: %v", err)
	}
	identityParams := dbgen.CreateLocalIdentityParams{
		OrganizationID:  organization.ID,
		UserID:          firstUser.ID,
		Email:           "pessoa@orgao.gov.br",
		EmailNormalized: "pessoa@orgao.gov.br",
		PasswordHash:    pgtype.Text{String: "hash", Valid: true},
	}
	if _, err := queries.CreateLocalIdentity(ctx, identityParams); err != nil {
		t.Fatalf("create first local identity: %v", err)
	}

	secondUser, err := queries.CreateUser(ctx, "Segunda Pessoa")
	if err != nil {
		t.Fatalf("create second user: %v", err)
	}
	identityParams.UserID = secondUser.ID
	identityParams.Email = "Pessoa@Orgao.Gov.Br"
	_, err = queries.CreateLocalIdentity(ctx, identityParams)
	if err == nil {
		t.Fatal("duplicate normalized local email was accepted")
	}
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) || postgresError.Code != "23505" {
		t.Fatalf("duplicate error = %v, want PostgreSQL unique violation", err)
	}

	authRepository := authdomain.PostgresRepository{
		Queries:          queries,
		OrganizationSlug: "orgao-teste",
	}
	account, err := authRepository.FindLocalAccount(ctx, "pessoa@orgao.gov.br")
	if err != nil || account.DisplayName != "Primeira Pessoa" {
		t.Fatalf("find local account = %#v, %v", account, err)
	}
	tokenHash := bytes.Repeat([]byte{0x55}, 32)
	if err := authRepository.CreateSession(ctx, authdomain.SessionRecord{
		UserID:    account.UserID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}); err != nil {
		t.Fatalf("create auth session: %v", err)
	}
	if _, err := authRepository.FindActiveSession(ctx, tokenHash); err != nil {
		t.Fatalf("find active session: %v", err)
	}
	if err := authRepository.RevokeSession(ctx, tokenHash); err != nil {
		t.Fatalf("revoke session: %v", err)
	}
	if _, err := authRepository.FindActiveSession(ctx, tokenHash); !errors.Is(err, authdomain.ErrSessionNotFound) {
		t.Fatalf("revoked session lookup error = %v", err)
	}
}
