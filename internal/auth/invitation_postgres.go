package auth

import (
	"context"
	"errors"
	"time"

	"github.com/etpeto/etpeto/internal/platform/database/dbgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresInvitationRepository needs the pool rather than the generated queries
// because accepting an invitation creates a user, an identity and a membership
// and burns the invitation, all in one transaction.
type PostgresInvitationRepository struct {
	Pool *pgxpool.Pool
}

func (r PostgresInvitationRepository) ListPeople(
	ctx context.Context,
	organizationID string,
) ([]Person, error) {
	parsed, err := parseUUID(organizationID)
	if err != nil {
		return nil, err
	}
	rows, err := dbgen.New(r.Pool).ListPeople(ctx, parsed)
	if err != nil {
		return nil, err
	}
	people := make([]Person, 0, len(rows))
	for _, row := range rows {
		person := Person{
			UserID:       formatUUID(row.UserID),
			InvitationID: formatUUID(row.InvitationID),
			DisplayName:  row.DisplayName,
			Email:        row.Email,
			Role:         row.Role,
			Pending:      row.Pending,
		}
		if row.InvitedUntil.Valid {
			person.InvitedUntil = row.InvitedUntil.Time
		}
		people = append(people, person)
	}
	return people, nil
}

// CreateInvitation replaces any live invitation for the same address, so a
// person never ends up with two usable links.
func (r PostgresInvitationRepository) CreateInvitation(
	ctx context.Context,
	invitation NewInvitation,
) (string, error) {
	organizationID, err := parseUUID(invitation.OrganizationID)
	if err != nil {
		return "", err
	}
	createdBy, err := parseUUID(invitation.CreatedBy)
	if err != nil {
		return "", err
	}

	transaction, err := r.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	queries := dbgen.New(r.Pool).WithTx(transaction)
	if _, err := queries.DeletePendingInvitation(ctx, dbgen.DeletePendingInvitationParams{
		OrganizationID:  organizationID,
		EmailNormalized: invitation.EmailNormalized,
	}); err != nil {
		return "", err
	}
	id, err := queries.CreateInvitation(ctx, dbgen.CreateInvitationParams{
		OrganizationID:  organizationID,
		Email:           invitation.Email,
		EmailNormalized: invitation.EmailNormalized,
		DisplayName:     invitation.DisplayName,
		Role:            invitation.Role,
		TokenHash:       invitation.TokenHash,
		ExpiresAt:       pgtype.Timestamptz{Time: invitation.ExpiresAt, Valid: true},
		CreatedBy:       createdBy,
	})
	if err != nil {
		return "", err
	}
	if err := transaction.Commit(ctx); err != nil {
		return "", err
	}
	return formatUUID(id), nil
}

func (r PostgresInvitationRepository) FindInvitationByTokenHash(
	ctx context.Context,
	tokenHash []byte,
) (PendingInvitation, error) {
	row, err := dbgen.New(r.Pool).FindInvitationByTokenHash(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return PendingInvitation{}, ErrInvitationNotFound
	}
	if err != nil {
		return PendingInvitation{}, err
	}
	return PendingInvitation{
		ID:              formatUUID(row.ID),
		OrganizationID:  formatUUID(row.OrganizationID),
		Email:           row.Email,
		EmailNormalized: row.EmailNormalized,
		DisplayName:     row.DisplayName,
		Role:            row.Role,
		ExpiresAt:       row.ExpiresAt.Time,
		AcceptedAt:      acceptedAt(row.AcceptedAt),
	}, nil
}

// AcceptInvitation is where the link is burned. Marking it used inside the same
// transaction that creates the account means two simultaneous acceptances
// cannot both succeed.
func (r PostgresInvitationRepository) AcceptInvitation(
	ctx context.Context,
	invitationID string,
	passwordHash string,
) (AcceptedInvitation, error) {
	parsedInvitation, err := parseUUID(invitationID)
	if err != nil {
		return AcceptedInvitation{}, ErrInvitationNotFound
	}

	transaction, err := r.Pool.Begin(ctx)
	if err != nil {
		return AcceptedInvitation{}, err
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	queries := dbgen.New(r.Pool).WithTx(transaction)
	invitation, err := queries.FindInvitationByID(ctx, parsedInvitation)
	if errors.Is(err, pgx.ErrNoRows) {
		return AcceptedInvitation{}, ErrInvitationNotFound
	}
	if err != nil {
		return AcceptedInvitation{}, err
	}
	if invitation.AcceptedAt.Valid {
		return AcceptedInvitation{}, ErrInvitationUsed
	}

	user, err := queries.CreateUser(ctx, invitation.DisplayName)
	if err != nil {
		return AcceptedInvitation{}, err
	}
	if _, err := queries.CreateLocalIdentity(ctx, dbgen.CreateLocalIdentityParams{
		OrganizationID:  invitation.OrganizationID,
		UserID:          user.ID,
		Email:           invitation.Email,
		EmailNormalized: invitation.EmailNormalized,
		PasswordHash:    pgtype.Text{String: passwordHash, Valid: true},
	}); err != nil {
		return AcceptedInvitation{}, err
	}
	if _, err := queries.CreateMembership(ctx, dbgen.CreateMembershipParams{
		OrganizationID: invitation.OrganizationID,
		UserID:         user.ID,
		Role:           invitation.Role,
	}); err != nil {
		return AcceptedInvitation{}, err
	}
	burned, err := queries.MarkInvitationAccepted(ctx, dbgen.MarkInvitationAcceptedParams{
		ID:             parsedInvitation,
		AcceptedUserID: user.ID,
	})
	if err != nil {
		return AcceptedInvitation{}, err
	}
	if burned == 0 {
		return AcceptedInvitation{}, ErrInvitationUsed
	}
	if err := transaction.Commit(ctx); err != nil {
		return AcceptedInvitation{}, err
	}
	return AcceptedInvitation{InvitationID: invitationID, UserID: formatUUID(user.ID)}, nil
}

func (r PostgresInvitationRepository) RevokeInvitation(
	ctx context.Context,
	organizationID string,
	invitationID string,
) error {
	parsedOrganization, err := parseUUID(organizationID)
	if err != nil {
		return err
	}
	parsedInvitation, err := parseUUID(invitationID)
	if err != nil {
		return ErrInvitationNotFound
	}
	removed, err := dbgen.New(r.Pool).RevokeInvitation(ctx, dbgen.RevokeInvitationParams{
		ID:             parsedInvitation,
		OrganizationID: parsedOrganization,
	})
	if err != nil {
		return err
	}
	if removed == 0 {
		return ErrInvitationNotFound
	}
	return nil
}

func (r PostgresInvitationRepository) ChangeRole(
	ctx context.Context,
	organizationID string,
	userID string,
	role string,
) error {
	parsedOrganization, err := parseUUID(organizationID)
	if err != nil {
		return err
	}
	parsedUser, err := parseUUID(userID)
	if err != nil {
		return ErrAccountNotFound
	}
	changed, err := dbgen.New(r.Pool).ChangeMembershipRole(ctx, dbgen.ChangeMembershipRoleParams{
		OrganizationID: parsedOrganization,
		UserID:         parsedUser,
		Role:           role,
	})
	if err != nil {
		return err
	}
	if changed == 0 {
		return ErrAccountNotFound
	}
	return nil
}

func (r PostgresInvitationRepository) HasLocalAccount(
	ctx context.Context,
	organizationID string,
	normalizedEmail string,
) (bool, error) {
	parsed, err := parseUUID(organizationID)
	if err != nil {
		return false, err
	}
	return dbgen.New(r.Pool).HasLocalAccount(ctx, dbgen.HasLocalAccountParams{
		OrganizationID:  parsed,
		EmailNormalized: normalizedEmail,
	})
}

func (r PostgresInvitationRepository) CountAdministrators(
	ctx context.Context,
	organizationID string,
) (int, error) {
	parsed, err := parseUUID(organizationID)
	if err != nil {
		return 0, err
	}
	total, err := dbgen.New(r.Pool).CountOrganizationAdministrators(ctx, parsed)
	return int(total), err
}

func acceptedAt(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}
