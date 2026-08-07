package auth

import (
	"bytes"
	"testing"
	"time"
)

func TestSessionIssuerCreatesOpaqueTokenAndAbsoluteExpiration(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	issuer := SessionIssuer{
		Random: stubRandom(0x7a, 0x1f),
		Now:    func() time.Time { return now },
		TTL:    12 * time.Hour,
	}

	session, err := issuer.Issue()
	if err != nil {
		t.Fatal(err)
	}
	if len(session.Token) != 43 {
		t.Fatalf("token length = %d, want 43 base64url characters", len(session.Token))
	}
	if session.ExpiresAt != now.Add(12*time.Hour) {
		t.Fatalf("expires at = %v", session.ExpiresAt)
	}
	if !MatchesTokenHash(session.Hash[:], session.Token) {
		t.Fatal("stored hash did not match issued token")
	}
	if MatchesTokenHash(session.Hash[:], session.Token+"x") {
		t.Fatal("different token matched stored hash")
	}
}

func TestSessionIssuerCreatesCSRFTokenBoundToTheSession(t *testing.T) {
	issuer := SessionIssuer{
		Random: stubRandom(0x7a, 0x1f),
		Now:    func() time.Time { return time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC) },
		TTL:    12 * time.Hour,
	}

	session, err := issuer.Issue()
	if err != nil {
		t.Fatal(err)
	}
	if session.CSRFToken == "" {
		t.Fatal("session was issued without a CSRF token")
	}
	if session.CSRFToken == session.Token {
		t.Fatal("CSRF token reused the session token")
	}
	if !MatchesTokenHash(session.CSRFHash[:], session.CSRFToken) {
		t.Fatal("stored CSRF hash did not match the issued CSRF token")
	}
	if MatchesTokenHash(session.CSRFHash[:], session.Token) {
		t.Fatal("session token validated against the CSRF hash")
	}
}

func stubRandom(sessionFill byte, csrfFill byte) *bytes.Reader {
	raw := append(
		bytes.Repeat([]byte{sessionFill}, sessionTokenBytes),
		bytes.Repeat([]byte{csrfFill}, sessionTokenBytes)...,
	)
	return bytes.NewReader(raw)
}

func TestSessionExpirationUsesAbsoluteDeadline(t *testing.T) {
	deadline := time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC)
	if SessionActive(deadline, deadline) {
		t.Fatal("session remained active at its absolute deadline")
	}
	if !SessionActive(deadline, deadline.Add(-time.Nanosecond)) {
		t.Fatal("session expired before its deadline")
	}
}

func TestHashTokenRejectsMalformedToken(t *testing.T) {
	if _, err := HashToken("not base64url!"); err == nil {
		t.Fatal("malformed token was accepted")
	}
}
