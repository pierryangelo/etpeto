package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/etpeto/etpeto/internal/platform/database/dbgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostgresRepository struct {
	Queries          *dbgen.Queries
	OrganizationSlug string
}

func (r PostgresRepository) FindLocalAccount(
	ctx context.Context,
	normalizedEmail string,
) (Account, error) {
	row, err := r.Queries.FindLocalAccountByNormalizedEmail(
		ctx,
		dbgen.FindLocalAccountByNormalizedEmailParams{
			Slug:            r.OrganizationSlug,
			EmailNormalized: normalizedEmail,
		},
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrAccountNotFound
	}
	if err != nil {
		return Account{}, err
	}
	if !row.PasswordHash.Valid {
		return Account{}, ErrAccountNotFound
	}
	return Account{
		UserID:       formatUUID(row.ID),
		DisplayName:  row.DisplayName,
		PasswordHash: row.PasswordHash.String,
	}, nil
}

func (r PostgresRepository) CreateSession(ctx context.Context, record SessionRecord) error {
	userID, err := parseUUID(record.UserID)
	if err != nil {
		return err
	}
	_, err = r.Queries.CreateSession(ctx, dbgen.CreateSessionParams{
		UserID:    userID,
		TokenHash: record.TokenHash,
		CsrfHash:  record.CSRFHash,
		ExpiresAt: pgtype.Timestamptz{Time: record.ExpiresAt, Valid: true},
	})
	return err
}

func (r PostgresRepository) FindActiveSession(
	ctx context.Context,
	tokenHash []byte,
) (AuthenticatedUser, error) {
	row, err := r.Queries.FindActiveSession(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return AuthenticatedUser{}, ErrSessionNotFound
	}
	if err != nil {
		return AuthenticatedUser{}, err
	}
	if !row.ExpiresAt.Valid {
		return AuthenticatedUser{}, ErrSessionNotFound
	}
	return AuthenticatedUser{
		ID:             formatUUID(row.ID),
		DisplayName:    row.DisplayName,
		OrganizationID: formatUUID(row.OrganizationID),
		ExpiresAt:      row.ExpiresAt.Time,
		CSRFHash:       row.CsrfHash,
	}, nil
}

func (r PostgresRepository) RevokeSession(ctx context.Context, tokenHash []byte) error {
	_, err := r.Queries.RevokeSession(ctx, tokenHash)
	return err
}

func formatUUID(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return uuid.UUID(value.Bytes).String()
}

func parseUUID(value string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("parse user ID: %w", err)
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}
