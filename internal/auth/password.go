package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	maxPasswordBytes  = 1024
	passwordSaltBytes = 16
	passwordHashBytes = 32
)

type PasswordParams struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
}

var DefaultPasswordParams = PasswordParams{
	MemoryKiB:   64 * 1024,
	Iterations:  3,
	Parallelism: 2,
}

type PasswordHasher struct {
	Params PasswordParams
	Random io.Reader
}

func (h PasswordHasher) Hash(password string) (string, error) {
	if len(password) == 0 || len(password) > maxPasswordBytes {
		return "", errors.New("password length is outside the accepted range")
	}
	params, err := normalizePasswordParams(h.Params)
	if err != nil {
		return "", err
	}
	random := h.Random
	if random == nil {
		random = rand.Reader
	}
	salt := make([]byte, passwordSaltBytes)
	if _, err := io.ReadFull(random, salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.MemoryKiB,
		params.Parallelism,
		passwordHashBytes,
	)
	encoding := base64.RawStdEncoding
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		params.MemoryKiB,
		params.Iterations,
		params.Parallelism,
		encoding.EncodeToString(salt),
		encoding.EncodeToString(hash),
	), nil
}

func (h PasswordHasher) Verify(encoded string, password string) (bool, error) {
	if len(password) == 0 || len(password) > maxPasswordBytes {
		return false, errors.New("password length is outside the accepted range")
	}
	params, salt, expected, err := parsePasswordHash(encoded)
	if err != nil {
		return false, err
	}
	actual := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.MemoryKiB,
		params.Parallelism,
		uint32(len(expected)),
	)
	return subtle.ConstantTimeCompare(expected, actual) == 1, nil
}

func parsePasswordHash(encoded string) (PasswordParams, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return PasswordParams{}, nil, nil, errors.New("invalid Argon2id hash format")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return PasswordParams{}, nil, nil, errors.New("unsupported Argon2id version")
	}
	params := PasswordParams{}
	if _, err := fmt.Sscanf(
		parts[3],
		"m=%d,t=%d,p=%d",
		&params.MemoryKiB,
		&params.Iterations,
		&params.Parallelism,
	); err != nil {
		return PasswordParams{}, nil, nil, errors.New("invalid Argon2id parameters")
	}
	if _, err := normalizePasswordParams(params); err != nil {
		return PasswordParams{}, nil, nil, err
	}
	encoding := base64.RawStdEncoding
	salt, err := encoding.DecodeString(parts[4])
	if err != nil || len(salt) != passwordSaltBytes {
		return PasswordParams{}, nil, nil, errors.New("invalid Argon2id salt")
	}
	hash, err := encoding.DecodeString(parts[5])
	if err != nil || len(hash) != passwordHashBytes {
		return PasswordParams{}, nil, nil, errors.New("invalid Argon2id digest")
	}
	return params, salt, hash, nil
}

func normalizePasswordParams(params PasswordParams) (PasswordParams, error) {
	if params == (PasswordParams{}) {
		params = DefaultPasswordParams
	}
	if params.MemoryKiB < 8*1024 || params.MemoryKiB > 256*1024 {
		return PasswordParams{}, errors.New("Argon2id memory is outside the accepted range")
	}
	if params.Iterations < 1 || params.Iterations > 10 {
		return PasswordParams{}, errors.New("Argon2id iterations are outside the accepted range")
	}
	if params.Parallelism < 1 || params.Parallelism > 16 {
		return PasswordParams{}, errors.New("Argon2id parallelism is outside the accepted range")
	}
	return params, nil
}
