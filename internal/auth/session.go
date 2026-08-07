package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"
)

const sessionTokenBytes = 32

type IssuedSession struct {
	Token     string
	Hash      [sha256.Size]byte
	ExpiresAt time.Time
}

type SessionIssuer struct {
	Random io.Reader
	Now    func() time.Time
	TTL    time.Duration
}

func (i SessionIssuer) Issue() (IssuedSession, error) {
	if i.TTL <= 0 {
		return IssuedSession{}, errors.New("session TTL must be positive")
	}
	random := i.Random
	if random == nil {
		random = rand.Reader
	}
	now := i.Now
	if now == nil {
		now = time.Now
	}
	raw := make([]byte, sessionTokenBytes)
	if _, err := io.ReadFull(random, raw); err != nil {
		return IssuedSession{}, fmt.Errorf("generate session token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return IssuedSession{
		Token:     token,
		Hash:      sha256.Sum256(raw),
		ExpiresAt: now().UTC().Add(i.TTL),
	}, nil
}

func HashToken(token string) ([sha256.Size]byte, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != sessionTokenBytes {
		return [sha256.Size]byte{}, errors.New("invalid session token")
	}
	return sha256.Sum256(raw), nil
}

func MatchesTokenHash(stored []byte, token string) bool {
	hash, err := HashToken(token)
	if err != nil || len(stored) != sha256.Size {
		return false
	}
	return subtle.ConstantTimeCompare(stored, hash[:]) == 1
}

func SessionActive(expiresAt time.Time, now time.Time) bool {
	return now.Before(expiresAt)
}
