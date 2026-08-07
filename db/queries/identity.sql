-- name: CreateOrganization :one
INSERT INTO organizations (slug, name)
VALUES ($1, $2)
RETURNING id, slug, name, created_at;

-- name: CreateUser :one
INSERT INTO users (display_name)
VALUES ($1)
RETURNING id, display_name, disabled_at, created_at;

-- name: CreateLocalIdentity :one
INSERT INTO identities (
    organization_id,
    user_id,
    provider,
    email,
    email_normalized,
    password_hash
)
VALUES ($1, $2, 'local', $3, $4, $5)
RETURNING id, organization_id, user_id, provider, issuer, subject,
          email, email_normalized, password_hash, created_at;

-- name: FindLocalIdentityByNormalizedEmail :one
SELECT id, organization_id, user_id, provider, issuer, subject,
       email, email_normalized, password_hash, created_at
FROM identities
WHERE organization_id = $1
  AND provider = 'local'
  AND email_normalized = $2;

-- name: FindOrganizationBySlug :one
SELECT id, slug, name, created_at
FROM organizations
WHERE slug = $1;

-- name: CountOrganizationAdministrators :one
SELECT count(*)
FROM memberships
WHERE organization_id = $1
  AND role IN ('installation_admin', 'organization_admin');

-- name: CreateMembership :one
INSERT INTO memberships (organization_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING id, organization_id, user_id, role, created_at;

-- name: CreateSession :one
INSERT INTO sessions (user_id, token_hash, csrf_hash, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, token_hash, csrf_hash, expires_at, revoked_at, created_at;

-- name: RevokeSession :execrows
UPDATE sessions
SET revoked_at = now()
WHERE token_hash = $1
  AND revoked_at IS NULL;

-- name: FindLocalAccountByNormalizedEmail :one
SELECT users.id, users.display_name, identities.password_hash
FROM identities
JOIN users ON users.id = identities.user_id
JOIN organizations ON organizations.id = identities.organization_id
WHERE organizations.slug = $1
  AND identities.provider = 'local'
  AND identities.email_normalized = $2
  AND users.disabled_at IS NULL;

-- name: FindActiveSession :one
SELECT users.id,
       users.display_name,
       sessions.expires_at,
       sessions.csrf_hash,
       memberships.organization_id
FROM sessions
JOIN users ON users.id = sessions.user_id
JOIN memberships ON memberships.user_id = users.id
WHERE sessions.token_hash = $1
  AND sessions.revoked_at IS NULL
  AND sessions.expires_at > now()
  AND users.disabled_at IS NULL;
