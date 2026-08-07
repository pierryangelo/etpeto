package auth

import (
	"context"
	"errors"

	"github.com/etpeto/etpeto/internal/platform/database/dbgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresProvisioningRepository writes the first administrator. It needs the
// pool rather than the generated queries because user, identity and membership
// must be created in a single transaction.
type PostgresProvisioningRepository struct {
	Pool *pgxpool.Pool
}

func (r PostgresProvisioningRepository) FindOrganizationBySlug(
	ctx context.Context,
	slug string,
) (string, error) {
	organization, err := dbgen.New(r.Pool).FindOrganizationBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrOrganizationNotFound
	}
	if err != nil {
		return "", err
	}
	return formatUUID(organization.ID), nil
}

func (r PostgresProvisioningRepository) CreateOrganization(
	ctx context.Context,
	organization NewOrganization,
) (string, error) {
	created, err := dbgen.New(r.Pool).CreateOrganization(ctx, dbgen.CreateOrganizationParams{
		Slug: organization.Slug,
		Name: organization.Name,
	})
	if err != nil {
		return "", err
	}
	return formatUUID(created.ID), nil
}

func (r PostgresProvisioningRepository) FindLocalUserID(
	ctx context.Context,
	organizationID string,
	normalizedEmail string,
) (string, error) {
	parsedOrganizationID, err := parseUUID(organizationID)
	if err != nil {
		return "", err
	}
	identity, err := dbgen.New(r.Pool).FindLocalIdentityByNormalizedEmail(
		ctx,
		dbgen.FindLocalIdentityByNormalizedEmailParams{
			OrganizationID:  parsedOrganizationID,
			EmailNormalized: normalizedEmail,
		},
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrAccountNotFound
	}
	if err != nil {
		return "", err
	}
	return formatUUID(identity.UserID), nil
}

func (r PostgresProvisioningRepository) CountAdministrators(
	ctx context.Context,
	organizationID string,
) (int, error) {
	parsedOrganizationID, err := parseUUID(organizationID)
	if err != nil {
		return 0, err
	}
	total, err := dbgen.New(r.Pool).CountOrganizationAdministrators(ctx, parsedOrganizationID)
	if err != nil {
		return 0, err
	}
	return int(total), nil
}

func (r PostgresProvisioningRepository) CreateAdministrator(
	ctx context.Context,
	administrator NewAdministrator,
) (string, error) {
	organizationID, err := parseUUID(administrator.OrganizationID)
	if err != nil {
		return "", err
	}
	transaction, err := r.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	queries := dbgen.New(r.Pool).WithTx(transaction)
	user, err := queries.CreateUser(ctx, administrator.DisplayName)
	if err != nil {
		return "", err
	}
	if _, err := queries.CreateLocalIdentity(ctx, dbgen.CreateLocalIdentityParams{
		OrganizationID:  organizationID,
		UserID:          user.ID,
		Email:           administrator.Email,
		EmailNormalized: administrator.EmailNormalized,
		PasswordHash:    pgtype.Text{String: administrator.PasswordHash, Valid: true},
	}); err != nil {
		return "", err
	}
	if _, err := queries.CreateMembership(ctx, dbgen.CreateMembershipParams{
		OrganizationID: organizationID,
		UserID:         user.ID,
		Role:           administrator.Role,
	}); err != nil {
		return "", err
	}
	if err := transaction.Commit(ctx); err != nil {
		return "", err
	}
	return formatUUID(user.ID), nil
}
