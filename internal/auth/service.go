package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrAccountNotFound    = errors.New("local account not found")
	ErrSessionNotFound    = errors.New("session not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidSession     = errors.New("invalid session")
)

type Account struct {
	UserID       string
	DisplayName  string
	PasswordHash string
}

type SessionRecord struct {
	UserID    string
	TokenHash []byte
	ExpiresAt time.Time
}

type AuthenticatedUser struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"displayName"`
	ExpiresAt   time.Time `json:"-"`
}

type LoginResult struct {
	Token     string
	ExpiresAt time.Time
	User      AuthenticatedUser
}

type Repository interface {
	FindLocalAccount(context.Context, string) (Account, error)
	CreateSession(context.Context, SessionRecord) error
	FindActiveSession(context.Context, []byte) (AuthenticatedUser, error)
	RevokeSession(context.Context, []byte) error
}

type Service struct {
	repository Repository
	passwords  PasswordHasher
	sessions   SessionIssuer
	dummyHash  string
	now        func() time.Time
}

func NewService(
	repository Repository,
	passwords PasswordHasher,
	sessions SessionIssuer,
) (*Service, error) {
	if repository == nil {
		return nil, errors.New("authentication repository is required")
	}
	dummyHash, err := passwords.Hash("etpeto-credential-timing-placeholder")
	if err != nil {
		return nil, fmt.Errorf("create credential timing hash: %w", err)
	}
	now := sessions.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		repository: repository,
		passwords:  passwords,
		sessions:   sessions,
		dummyHash:  dummyHash,
		now:        now,
	}, nil
}

func (s *Service) Login(
	ctx context.Context,
	email string,
	password string,
) (LoginResult, error) {
	normalizedEmail := NormalizeEmail(email)
	if normalizedEmail == "" {
		_, _ = s.passwords.Verify(s.dummyHash, password)
		return LoginResult{}, ErrInvalidCredentials
	}

	account, err := s.repository.FindLocalAccount(ctx, normalizedEmail)
	if errors.Is(err, ErrAccountNotFound) {
		_, _ = s.passwords.Verify(s.dummyHash, password)
		return LoginResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, fmt.Errorf("find local account: %w", err)
	}
	verified, err := s.passwords.Verify(account.PasswordHash, password)
	if err != nil || !verified {
		return LoginResult{}, ErrInvalidCredentials
	}

	issued, err := s.sessions.Issue()
	if err != nil {
		return LoginResult{}, err
	}
	if err := s.repository.CreateSession(ctx, SessionRecord{
		UserID:    account.UserID,
		TokenHash: issued.Hash[:],
		ExpiresAt: issued.ExpiresAt,
	}); err != nil {
		return LoginResult{}, fmt.Errorf("create session: %w", err)
	}
	return LoginResult{
		Token:     issued.Token,
		ExpiresAt: issued.ExpiresAt,
		User: AuthenticatedUser{
			ID:          account.UserID,
			DisplayName: account.DisplayName,
			ExpiresAt:   issued.ExpiresAt,
		},
	}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (AuthenticatedUser, error) {
	hash, err := HashToken(token)
	if err != nil {
		return AuthenticatedUser{}, ErrInvalidSession
	}
	user, err := s.repository.FindActiveSession(ctx, hash[:])
	if errors.Is(err, ErrSessionNotFound) {
		return AuthenticatedUser{}, ErrInvalidSession
	}
	if err != nil {
		return AuthenticatedUser{}, fmt.Errorf("find session: %w", err)
	}
	if !SessionActive(user.ExpiresAt, s.now().UTC()) {
		return AuthenticatedUser{}, ErrInvalidSession
	}
	return user, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	hash, err := HashToken(token)
	if err != nil {
		return nil
	}
	if err := s.repository.RevokeSession(ctx, hash[:]); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
