package auth

import (
	"bytes"
	"testing"
	"time"
)

func TestSessionIssuerCreatesOpaqueTokenAndAbsoluteExpiration(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	issuer := SessionIssuer{
		Random: bytes.NewReader(bytes.Repeat([]byte{0x7a}, sessionTokenBytes)),
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
