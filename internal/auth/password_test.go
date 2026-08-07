package auth

import (
	"bytes"
	"strings"
	"testing"
)

func TestPasswordHasherRoundTrip(t *testing.T) {
	hasher := PasswordHasher{
		Params: PasswordParams{MemoryKiB: 8 * 1024, Iterations: 1, Parallelism: 1},
		Random: bytes.NewReader(bytes.Repeat([]byte{0x42}, passwordSaltBytes)),
	}

	encoded, err := hasher.Hash("uma senha longa e segura")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$v=19$") {
		t.Fatalf("encoded hash = %q", encoded)
	}
	verified, err := hasher.Verify(encoded, "uma senha longa e segura")
	if err != nil || !verified {
		t.Fatalf("Verify() = %v, %v", verified, err)
	}
}

func TestPasswordHasherRejectsWrongPassword(t *testing.T) {
	hasher := testPasswordHasher()
	encoded, err := hasher.Hash("senha correta")
	if err != nil {
		t.Fatal(err)
	}

	verified, err := hasher.Verify(encoded, "senha incorreta")
	if err != nil {
		t.Fatal(err)
	}
	if verified {
		t.Fatal("wrong password was accepted")
	}
}

func TestPasswordHasherRejectsMalformedHash(t *testing.T) {
	verified, err := testPasswordHasher().Verify("not-a-phc-hash", "senha")
	if err == nil || verified {
		t.Fatalf("Verify() = %v, %v; want malformed hash error", verified, err)
	}
}

func TestPasswordHasherLimitsPasswordBytes(t *testing.T) {
	hasher := testPasswordHasher()
	_, err := hasher.Hash(strings.Repeat("x", maxPasswordBytes+1))
	if err == nil {
		t.Fatal("oversized password was accepted")
	}
}

func testPasswordHasher() PasswordHasher {
	return PasswordHasher{
		Params: PasswordParams{MemoryKiB: 8 * 1024, Iterations: 1, Parallelism: 1},
		Random: bytes.NewReader(bytes.Repeat([]byte{0x24}, passwordSaltBytes*8)),
	}
}
