package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Roles recognised by the installation. Provisioning only ever grants the
// organization administrator role; everything above it is a future decision.
const (
	RoleInstallationAdmin = "installation_admin"
	RoleOrganizationAdmin = "organization_admin"
	RoleTemplateManager   = "template_manager"
	RoleAuthor            = "author"
)

// minimumProvisionedPasswordLength is a floor for the very first credential of
// an installation, which is typed once and distributed by an institutional
// channel. It is not the full password policy, which is still pending.
const minimumProvisionedPasswordLength = 12

var (
	ErrOrganizationNotFound           = errors.New("organization not found")
	ErrInvalidProvisionRequest        = errors.New("invalid provisioning request")
	ErrWeakPassword                   = errors.New("password is shorter than the accepted floor")
	ErrInstallationAlreadyInitialized = errors.New("installation already has an administrator")
)

type NewOrganization struct {
	Slug string
	Name string
}

type NewAdministrator struct {
	OrganizationID  string
	DisplayName     string
	Email           string
	EmailNormalized string
	PasswordHash    string
	Role            string
}

type ProvisionRequest struct {
	OrganizationSlug string
	OrganizationName string
	Email            string
	DisplayName      string
	Password         string
}

type ProvisionOutcome struct {
	Created        bool
	OrganizationID string
	UserID         string
}

type ProvisioningRepository interface {
	FindOrganizationBySlug(context.Context, string) (string, error)
	CreateOrganization(context.Context, NewOrganization) (string, error)
	FindLocalUserID(ctx context.Context, organizationID string, normalizedEmail string) (string, error)
	CountAdministrators(context.Context, string) (int, error)
	CreateAdministrator(context.Context, NewAdministrator) (string, error)
}

// Provisioner creates the first local administrator of an installation. It is
// idempotent for the same address and refuses to add a second administrator to
// an installation that already has one, so re-running it can never silently
// widen access.
type Provisioner struct {
	Repository ProvisioningRepository
	Passwords  PasswordHasher
}

func (p Provisioner) Provision(
	ctx context.Context,
	request ProvisionRequest,
) (ProvisionOutcome, error) {
	normalized, err := normalizeProvisionRequest(request)
	if err != nil {
		return ProvisionOutcome{}, err
	}

	organizationID, err := p.Repository.FindOrganizationBySlug(ctx, normalized.OrganizationSlug)
	if errors.Is(err, ErrOrganizationNotFound) {
		organizationID, err = p.Repository.CreateOrganization(ctx, NewOrganization{
			Slug: normalized.OrganizationSlug,
			Name: normalized.OrganizationName,
		})
		if err != nil {
			return ProvisionOutcome{}, fmt.Errorf("create organization: %w", err)
		}
	} else if err != nil {
		return ProvisionOutcome{}, fmt.Errorf("find organization: %w", err)
	}

	normalizedEmail := NormalizeEmail(normalized.Email)
	userID, err := p.Repository.FindLocalUserID(ctx, organizationID, normalizedEmail)
	if err == nil {
		return ProvisionOutcome{OrganizationID: organizationID, UserID: userID}, nil
	}
	if !errors.Is(err, ErrAccountNotFound) {
		return ProvisionOutcome{}, fmt.Errorf("find local account: %w", err)
	}

	administrators, err := p.Repository.CountAdministrators(ctx, organizationID)
	if err != nil {
		return ProvisionOutcome{}, fmt.Errorf("count administrators: %w", err)
	}
	if administrators > 0 {
		return ProvisionOutcome{}, ErrInstallationAlreadyInitialized
	}

	passwordHash, err := p.Passwords.Hash(normalized.Password)
	if err != nil {
		return ProvisionOutcome{}, fmt.Errorf("hash password: %w", err)
	}
	userID, err = p.Repository.CreateAdministrator(ctx, NewAdministrator{
		OrganizationID:  organizationID,
		DisplayName:     normalized.DisplayName,
		Email:           normalized.Email,
		EmailNormalized: normalizedEmail,
		PasswordHash:    passwordHash,
		Role:            RoleOrganizationAdmin,
	})
	if err != nil {
		return ProvisionOutcome{}, fmt.Errorf("create administrator: %w", err)
	}
	return ProvisionOutcome{Created: true, OrganizationID: organizationID, UserID: userID}, nil
}

func normalizeProvisionRequest(request ProvisionRequest) (ProvisionRequest, error) {
	normalized := ProvisionRequest{
		OrganizationSlug: strings.TrimSpace(request.OrganizationSlug),
		OrganizationName: strings.TrimSpace(request.OrganizationName),
		Email:            strings.TrimSpace(request.Email),
		DisplayName:      strings.TrimSpace(request.DisplayName),
		Password:         request.Password,
	}
	if normalized.OrganizationSlug == "" || normalized.Email == "" || normalized.DisplayName == "" {
		return ProvisionRequest{}, ErrInvalidProvisionRequest
	}
	if !strings.Contains(normalized.Email, "@") {
		return ProvisionRequest{}, ErrInvalidProvisionRequest
	}
	if normalized.OrganizationName == "" {
		normalized.OrganizationName = normalized.OrganizationSlug
	}
	if utf8.RuneCountInString(normalized.Password) < minimumProvisionedPasswordLength {
		return ProvisionRequest{}, ErrWeakPassword
	}
	return normalized, nil
}
