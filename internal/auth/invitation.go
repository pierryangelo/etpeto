package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// InvitationTTL is how long a link stays usable. A forgotten invitation in an
// inbox must not remain an open door months later.
const InvitationTTL = 72 * time.Hour

const maxDisplayNameRunes = 200

var (
	ErrInvitationNotFound   = errors.New("invitation not found")
	ErrInvitationUsed       = errors.New("invitation already accepted")
	ErrInvitationExpired    = errors.New("invitation expired")
	ErrInvalidInvitation    = errors.New("invalid invitation request")
	ErrAccountAlreadyExists = errors.New("address already has an account")
	ErrUnsupportedRole      = errors.New("unsupported role")
)

var assignableRoles = map[string]bool{
	RoleOrganizationAdmin: true,
	RoleTemplateManager:   true,
	RoleAuthor:            true,
}

// Person is a row of the access screen. A pending invitation appears alongside
// the people who already signed in, because from the administration's point of
// view both are "someone who was given access".
type Person struct {
	UserID       string    `json:"userId,omitempty"`
	InvitationID string    `json:"invitationId,omitempty"`
	DisplayName  string    `json:"displayName"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	Pending      bool      `json:"pending"`
	InvitedUntil time.Time `json:"invitedUntil,omitempty"`
}

type NewInvitation struct {
	OrganizationID  string
	Email           string
	EmailNormalized string
	DisplayName     string
	Role            string
	TokenHash       []byte
	ExpiresAt       time.Time
	CreatedBy       string
}

type PendingInvitation struct {
	ID              string
	OrganizationID  string
	Email           string
	EmailNormalized string
	DisplayName     string
	Role            string
	ExpiresAt       time.Time
	AcceptedAt      time.Time
}

type AcceptedInvitation struct {
	InvitationID string
	UserID       string
}

type InvitationRepository interface {
	ListPeople(ctx context.Context, organizationID string) ([]Person, error)
	CreateInvitation(context.Context, NewInvitation) (string, error)
	FindInvitationByTokenHash(ctx context.Context, tokenHash []byte) (PendingInvitation, error)
	// AcceptInvitation creates the user, the local identity and the membership,
	// and marks the invitation used, in a single transaction.
	AcceptInvitation(ctx context.Context, invitationID string, passwordHash string) (AcceptedInvitation, error)
	RevokeInvitation(ctx context.Context, organizationID string, invitationID string) error
	ChangeRole(ctx context.Context, organizationID string, userID string, role string) error
	HasLocalAccount(ctx context.Context, organizationID string, normalizedEmail string) (bool, error)
	CountAdministrators(ctx context.Context, organizationID string) (int, error)
}

// Invitations issues single-use links and turns them into accounts.
//
// The invitation carries no password: the person sets their own when accepting,
// so the administration never knows it. Only the hash of the token reaches the
// database, exactly as with a session, and accepting burns it.
type Invitations struct {
	Repository InvitationRepository
	Passwords  PasswordHasher
	Tokens     SessionIssuer
	Now        func() time.Time
}

func (i Invitations) now() time.Time {
	if i.Now != nil {
		return i.Now().UTC()
	}
	return time.Now().UTC()
}

func (i Invitations) List(ctx context.Context, organizationID string) ([]Person, error) {
	return i.Repository.ListPeople(ctx, organizationID)
}

// Invite creates the invitation and returns the raw token exactly once. It is
// never stored and never shown again: losing it means issuing a new invitation.
func (i Invitations) Invite(
	ctx context.Context,
	organizationID string,
	createdBy string,
	email string,
	displayName string,
	role string,
) (string, Person, error) {
	trimmedName := strings.TrimSpace(displayName)
	normalizedEmail := NormalizeEmail(email)
	if normalizedEmail == "" || !strings.Contains(normalizedEmail, "@") {
		return "", Person{}, fmt.Errorf("%w: email", ErrInvalidInvitation)
	}
	if trimmedName == "" || utf8.RuneCountInString(trimmedName) > maxDisplayNameRunes {
		return "", Person{}, fmt.Errorf("%w: display name", ErrInvalidInvitation)
	}
	if !assignableRoles[role] {
		return "", Person{}, ErrUnsupportedRole
	}

	taken, err := i.Repository.HasLocalAccount(ctx, organizationID, normalizedEmail)
	if err != nil {
		return "", Person{}, err
	}
	if taken {
		return "", Person{}, ErrAccountAlreadyExists
	}

	issuer := i.Tokens
	if issuer.TTL <= 0 {
		issuer.TTL = InvitationTTL
	}
	issued, err := issuer.Issue()
	if err != nil {
		return "", Person{}, err
	}
	expiresAt := i.now().Add(InvitationTTL)
	id, err := i.Repository.CreateInvitation(ctx, NewInvitation{
		OrganizationID:  organizationID,
		Email:           strings.TrimSpace(email),
		EmailNormalized: normalizedEmail,
		DisplayName:     trimmedName,
		Role:            role,
		TokenHash:       issued.Hash[:],
		ExpiresAt:       expiresAt,
		CreatedBy:       createdBy,
	})
	if err != nil {
		return "", Person{}, err
	}
	return issued.Token, Person{
		InvitationID: id,
		DisplayName:  trimmedName,
		Email:        strings.TrimSpace(email),
		Role:         role,
		Pending:      true,
		InvitedUntil: expiresAt,
	}, nil
}

// Accept turns a valid token into an account with the password the person
// chose. The invitation is burned in the same transaction, so a leaked link is
// useless once used.
func (i Invitations) Accept(
	ctx context.Context,
	token string,
	password string,
) (AcceptedInvitation, error) {
	invitation, err := i.resolve(ctx, token)
	if err != nil {
		return AcceptedInvitation{}, err
	}
	if utf8.RuneCountInString(password) < minimumProvisionedPasswordLength {
		return AcceptedInvitation{}, ErrWeakPassword
	}
	passwordHash, err := i.Passwords.Hash(password)
	if err != nil {
		return AcceptedInvitation{}, fmt.Errorf("hash password: %w", err)
	}
	return i.Repository.AcceptInvitation(ctx, invitation.ID, passwordHash)
}

// Describe reports who an invitation is for, so the acceptance page can greet
// the person without them having to type their address again.
func (i Invitations) Describe(
	ctx context.Context,
	token string,
) (PendingInvitation, error) {
	return i.resolve(ctx, token)
}

func (i Invitations) resolve(
	ctx context.Context,
	token string,
) (PendingInvitation, error) {
	hash, err := HashToken(token)
	if err != nil {
		return PendingInvitation{}, ErrInvitationNotFound
	}
	invitation, err := i.Repository.FindInvitationByTokenHash(ctx, hash[:])
	if err != nil {
		return PendingInvitation{}, err
	}
	if !invitation.AcceptedAt.IsZero() {
		return PendingInvitation{}, ErrInvitationUsed
	}
	if !i.now().Before(invitation.ExpiresAt) {
		return PendingInvitation{}, ErrInvitationExpired
	}
	return invitation, nil
}

func (i Invitations) Revoke(
	ctx context.Context,
	organizationID string,
	invitationID string,
) error {
	return i.Repository.RevokeInvitation(ctx, organizationID, invitationID)
}

// ChangeRole refuses to remove the last administrator: an organization that
// locks itself out has no way back short of touching the database.
func (i Invitations) ChangeRole(
	ctx context.Context,
	organizationID string,
	userID string,
	currentRole string,
	role string,
) error {
	if !assignableRoles[role] {
		return ErrUnsupportedRole
	}
	if isAdministrative(currentRole) && !isAdministrative(role) {
		administrators, err := i.Repository.CountAdministrators(ctx, organizationID)
		if err != nil {
			return err
		}
		if administrators <= 1 {
			return ErrInstallationAlreadyInitialized
		}
	}
	return i.Repository.ChangeRole(ctx, organizationID, userID, role)
}

func isAdministrative(role string) bool {
	return role == RoleOrganizationAdmin || role == RoleInstallationAdmin
}
