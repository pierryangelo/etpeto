package auth

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

func TestServiceLoginNormalizesEmailAndPersistsOnlyTokenHash(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	hasher := testPasswordHasher()
	passwordHash, err := hasher.Hash("senha correta")
	if err != nil {
		t.Fatal(err)
	}
	repository := &repositoryStub{account: Account{
		UserID:       "00000000-0000-0000-0000-000000000001",
		DisplayName:  "Pessoa Autora",
		PasswordHash: passwordHash,
	}}
	service := newTestService(t, repository, hasher, now)

	result, err := service.Login(context.Background(), "  PESSOA@ORGAO.GOV.BR ", "senha correta")
	if err != nil {
		t.Fatal(err)
	}
	if repository.email != "pessoa@orgao.gov.br" {
		t.Fatalf("normalized email = %q", repository.email)
	}
	if repository.created.TokenHash == nil || len(repository.created.TokenHash) != 32 {
		t.Fatalf("stored token hash = %x", repository.created.TokenHash)
	}
	if bytes.Contains(repository.created.TokenHash, []byte(result.Token)) {
		t.Fatal("raw token was persisted")
	}
	if result.User.DisplayName != "Pessoa Autora" || result.ExpiresAt != now.Add(time.Hour) {
		t.Fatalf("login result = %#v", result)
	}
}

func TestServiceLoginUsesOneGenericCredentialError(t *testing.T) {
	hasher := testPasswordHasher()
	passwordHash, err := hasher.Hash("senha correta")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		repository *repositoryStub
	}{
		{name: "missing user", repository: &repositoryStub{findErr: ErrAccountNotFound}},
		{name: "wrong password", repository: &repositoryStub{account: Account{PasswordHash: passwordHash}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := newTestService(t, test.repository, hasher, time.Now())
			_, err := service.Login(context.Background(), "pessoa@orgao.gov.br", "senha incorreta")
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf("Login() error = %v", err)
			}
			if test.repository.createCalls != 0 {
				t.Fatal("session was created before successful verification")
			}
		})
	}
}

func TestServiceAuthenticatesAndRevokesCurrentSession(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	issuer := SessionIssuer{
		Random: bytes.NewReader(bytes.Repeat([]byte{0x33}, sessionTokenBytes)),
		Now:    func() time.Time { return now },
		TTL:    time.Hour,
	}
	issued, err := issuer.Issue()
	if err != nil {
		t.Fatal(err)
	}
	repository := &repositoryStub{session: AuthenticatedUser{
		ID:          "00000000-0000-0000-0000-000000000001",
		DisplayName: "Pessoa Autora",
		ExpiresAt:   now.Add(time.Hour),
	}}
	service := newTestService(t, repository, testPasswordHasher(), now)

	user, err := service.Authenticate(context.Background(), issued.Token)
	if err != nil || user.DisplayName != "Pessoa Autora" {
		t.Fatalf("Authenticate() = %#v, %v", user, err)
	}
	if err := service.Logout(context.Background(), issued.Token); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(repository.revokedHash, issued.Hash[:]) {
		t.Fatalf("revoked hash = %x, want %x", repository.revokedHash, issued.Hash)
	}
}

func newTestService(
	t *testing.T,
	repository Repository,
	hasher PasswordHasher,
	now time.Time,
) *Service {
	t.Helper()
	service, err := NewService(repository, hasher, SessionIssuer{
		Random: bytes.NewReader(bytes.Repeat([]byte{0x19}, sessionTokenBytes*8)),
		Now:    func() time.Time { return now },
		TTL:    time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

type repositoryStub struct {
	account       Account
	findErr       error
	email         string
	created       SessionRecord
	createCalls   int
	session       AuthenticatedUser
	sessionErr    error
	revokedHash   []byte
	revocationErr error
}

func (r *repositoryStub) FindLocalAccount(_ context.Context, normalizedEmail string) (Account, error) {
	r.email = normalizedEmail
	return r.account, r.findErr
}

func (r *repositoryStub) CreateSession(_ context.Context, record SessionRecord) error {
	r.createCalls++
	r.created = record
	return nil
}

func (r *repositoryStub) FindActiveSession(_ context.Context, _ []byte) (AuthenticatedUser, error) {
	return r.session, r.sessionErr
}

func (r *repositoryStub) RevokeSession(_ context.Context, tokenHash []byte) error {
	r.revokedHash = append([]byte(nil), tokenHash...)
	return r.revocationErr
}
