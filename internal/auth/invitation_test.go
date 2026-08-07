package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

type invitationRepositoryStub struct {
	people         []Person
	created        NewInvitation
	createCalls    int
	pending        PendingInvitation
	findErr        error
	accepted       string
	acceptedHash   string
	acceptCalls    int
	hasAccount     bool
	administrators int
	changedRole    string
	changeCalls    int
}

func (r *invitationRepositoryStub) ListPeople(context.Context, string) ([]Person, error) {
	return r.people, nil
}

func (r *invitationRepositoryStub) CreateInvitation(_ context.Context, invitation NewInvitation) (string, error) {
	r.createCalls++
	r.created = invitation
	return "invitation-1", nil
}

func (r *invitationRepositoryStub) FindInvitationByTokenHash(context.Context, []byte) (PendingInvitation, error) {
	return r.pending, r.findErr
}

func (r *invitationRepositoryStub) AcceptInvitation(_ context.Context, invitationID string, passwordHash string) (AcceptedInvitation, error) {
	r.acceptCalls++
	r.accepted = invitationID
	r.acceptedHash = passwordHash
	return AcceptedInvitation{InvitationID: invitationID, UserID: "user-1"}, nil
}

func (r *invitationRepositoryStub) RevokeInvitation(context.Context, string, string) error {
	return nil
}

func (r *invitationRepositoryStub) ChangeRole(_ context.Context, _ string, _ string, role string) error {
	r.changeCalls++
	r.changedRole = role
	return nil
}

func (r *invitationRepositoryStub) HasLocalAccount(context.Context, string, string) (bool, error) {
	return r.hasAccount, nil
}

func (r *invitationRepositoryStub) CountAdministrators(context.Context, string) (int, error) {
	return r.administrators, nil
}

func testInvitations(repository InvitationRepository, now time.Time) Invitations {
	return Invitations{
		Repository: repository,
		Passwords:  testPasswordHasher(),
		Tokens:     SessionIssuer{Random: stubRandom(0x41, 0x42), TTL: InvitationTTL},
		Now:        func() time.Time { return now },
	}
}

func TestInviteStoresOnlyTheTokenHashAndReturnsTheLinkOnce(t *testing.T) {
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	repository := &invitationRepositoryStub{}
	invitations := testInvitations(repository, now)

	token, person, err := invitations.Invite(
		context.Background(), "org-1", "admin-1", " Pessoa@Orgao.Gov.BR ", " Nova Pessoa ", RoleAuthor,
	)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("no token was returned to be sent by the institutional channel")
	}
	// Only the hash is persisted, exactly as with a session token.
	if !MatchesTokenHash(repository.created.TokenHash, token) {
		t.Fatalf("stored hash does not match the issued token")
	}
	if string(repository.created.TokenHash) == token {
		t.Fatal("the raw token was persisted")
	}
	if repository.created.EmailNormalized != "pessoa@orgao.gov.br" {
		t.Fatalf("normalized email = %q", repository.created.EmailNormalized)
	}
	if repository.created.DisplayName != "Nova Pessoa" {
		t.Fatalf("display name = %q", repository.created.DisplayName)
	}
	if repository.created.ExpiresAt != now.Add(InvitationTTL) {
		t.Fatalf("expiry = %v", repository.created.ExpiresAt)
	}
	if !person.Pending || person.InvitationID != "invitation-1" {
		t.Fatalf("person = %#v", person)
	}
}

func TestInviteRefusesAnAddressThatAlreadyHasAnAccount(t *testing.T) {
	repository := &invitationRepositoryStub{hasAccount: true}
	invitations := testInvitations(repository, time.Now())

	_, _, err := invitations.Invite(
		context.Background(), "org-1", "admin-1", "pessoa@orgao.gov.br", "Pessoa", RoleAuthor,
	)

	if !errors.Is(err, ErrAccountAlreadyExists) {
		t.Fatalf("Invite() error = %v, want ErrAccountAlreadyExists", err)
	}
	if repository.createCalls != 0 {
		t.Fatal("an invitation was created for an existing account")
	}
}

func TestInviteValidatesBeforeTouchingTheDatabase(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		displayName string
		role        string
		wantErr     error
	}{
		{"blank email", "  ", "Pessoa", RoleAuthor, ErrInvalidInvitation},
		{"malformed email", "pessoa", "Pessoa", RoleAuthor, ErrInvalidInvitation},
		{"blank name", "pessoa@orgao.gov.br", " ", RoleAuthor, ErrInvalidInvitation},
		{"unknown role", "pessoa@orgao.gov.br", "Pessoa", "superuser", ErrUnsupportedRole},
		{
			name:  "installation admin is not assignable from the screen",
			email: "pessoa@orgao.gov.br", displayName: "Pessoa",
			role: RoleInstallationAdmin, wantErr: ErrUnsupportedRole,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &invitationRepositoryStub{}
			invitations := testInvitations(repository, time.Now())

			_, _, err := invitations.Invite(
				context.Background(), "org-1", "admin-1", test.email, test.displayName, test.role,
			)

			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Invite() error = %v, want %v", err, test.wantErr)
			}
			if repository.createCalls != 0 {
				t.Fatal("an invalid invitation reached the database")
			}
		})
	}
}

func pendingFixture(expiresAt time.Time) PendingInvitation {
	return PendingInvitation{
		ID:              "invitation-1",
		OrganizationID:  "org-1",
		Email:           "pessoa@orgao.gov.br",
		EmailNormalized: "pessoa@orgao.gov.br",
		DisplayName:     "Nova Pessoa",
		Role:            RoleAuthor,
		ExpiresAt:       expiresAt,
	}
}

func TestAcceptCreatesTheAccountWithThePersonsOwnPassword(t *testing.T) {
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	repository := &invitationRepositoryStub{pending: pendingFixture(now.Add(time.Hour))}
	invitations := testInvitations(repository, now)
	token := issuedToken(t)

	accepted, err := invitations.Accept(context.Background(), token, "senha escolhida por mim")
	if err != nil {
		t.Fatal(err)
	}
	if accepted.UserID != "user-1" || repository.accepted != "invitation-1" {
		t.Fatalf("accepted = %#v", accepted)
	}
	// The administration never learns the password: only its hash is stored.
	verified, err := testPasswordHasher().Verify(repository.acceptedHash, "senha escolhida por mim")
	if err != nil || !verified {
		t.Fatalf("stored hash does not verify the chosen password: %v", err)
	}
}

func TestAcceptRefusesAnInvitationAlreadyUsed(t *testing.T) {
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	used := pendingFixture(now.Add(time.Hour))
	used.AcceptedAt = now.Add(-time.Minute)
	repository := &invitationRepositoryStub{pending: used}
	invitations := testInvitations(repository, now)

	_, err := invitations.Accept(context.Background(), issuedToken(t), "senha escolhida por mim")

	// A link that leaks after being used must be worthless.
	if !errors.Is(err, ErrInvitationUsed) {
		t.Fatalf("Accept() error = %v, want ErrInvitationUsed", err)
	}
	if repository.acceptCalls != 0 {
		t.Fatal("a used invitation created a second account")
	}
}

func TestAcceptRefusesAnExpiredInvitation(t *testing.T) {
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	repository := &invitationRepositoryStub{pending: pendingFixture(now.Add(-time.Second))}
	invitations := testInvitations(repository, now)

	_, err := invitations.Accept(context.Background(), issuedToken(t), "senha escolhida por mim")

	if !errors.Is(err, ErrInvitationExpired) {
		t.Fatalf("Accept() error = %v, want ErrInvitationExpired", err)
	}
}

func TestAcceptRefusesAMalformedToken(t *testing.T) {
	invitations := testInvitations(&invitationRepositoryStub{}, time.Now())

	_, err := invitations.Accept(context.Background(), "não é um token", "senha escolhida por mim")

	if !errors.Is(err, ErrInvitationNotFound) {
		t.Fatalf("Accept() error = %v, want ErrInvitationNotFound", err)
	}
}

func TestAcceptEnforcesThePasswordFloor(t *testing.T) {
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	repository := &invitationRepositoryStub{pending: pendingFixture(now.Add(time.Hour))}
	invitations := testInvitations(repository, now)

	_, err := invitations.Accept(context.Background(), issuedToken(t), "curta")

	if !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("Accept() error = %v, want ErrWeakPassword", err)
	}
	if repository.acceptCalls != 0 {
		t.Fatal("a weak password created an account")
	}
}

func TestChangeRoleRefusesToRemoveTheLastAdministrator(t *testing.T) {
	repository := &invitationRepositoryStub{administrators: 1}
	invitations := testInvitations(repository, time.Now())

	err := invitations.ChangeRole(
		context.Background(), "org-1", "user-1", RoleOrganizationAdmin, RoleAuthor,
	)

	// An organization that locks itself out has no way back short of the database.
	if !errors.Is(err, ErrInstallationAlreadyInitialized) {
		t.Fatalf("ChangeRole() error = %v", err)
	}
	if repository.changeCalls != 0 {
		t.Fatal("the last administrator was demoted")
	}
}

func TestChangeRoleAllowsDemotionWhileAnotherAdministratorRemains(t *testing.T) {
	repository := &invitationRepositoryStub{administrators: 2}
	invitations := testInvitations(repository, time.Now())

	if err := invitations.ChangeRole(
		context.Background(), "org-1", "user-1", RoleOrganizationAdmin, RoleAuthor,
	); err != nil {
		t.Fatal(err)
	}
	if repository.changedRole != RoleAuthor {
		t.Fatalf("changed role = %q", repository.changedRole)
	}
}

func TestChangeRoleRejectsAnUnknownRole(t *testing.T) {
	repository := &invitationRepositoryStub{administrators: 3}
	invitations := testInvitations(repository, time.Now())

	err := invitations.ChangeRole(context.Background(), "org-1", "user-1", RoleAuthor, "superuser")

	if !errors.Is(err, ErrUnsupportedRole) {
		t.Fatalf("ChangeRole() error = %v, want ErrUnsupportedRole", err)
	}
}

// issuedToken mints the same token the stubbed issuer produces, so the test can
// present it as a browser would.
func issuedToken(t *testing.T) string {
	t.Helper()
	issued, err := SessionIssuer{Random: stubRandom(0x41, 0x42), TTL: InvitationTTL}.Issue()
	if err != nil {
		t.Fatal(err)
	}
	return issued.Token
}
