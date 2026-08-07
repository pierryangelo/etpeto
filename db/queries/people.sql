-- name: ListPeople :many
SELECT users.id AS user_id,
       NULL::uuid AS invitation_id,
       users.display_name,
       identities.email,
       memberships.role,
       false AS pending,
       NULL::timestamptz AS invited_until
FROM memberships
JOIN users ON users.id = memberships.user_id
JOIN identities
  ON identities.user_id = users.id
 AND identities.organization_id = memberships.organization_id
 AND identities.provider = 'local'
WHERE memberships.organization_id = $1
UNION ALL
SELECT NULL::uuid AS user_id,
       invitations.id AS invitation_id,
       invitations.display_name,
       invitations.email,
       invitations.role,
       true AS pending,
       invitations.expires_at AS invited_until
FROM invitations
WHERE invitations.organization_id = $1
  AND invitations.accepted_at IS NULL
ORDER BY pending DESC, display_name;

-- Replacing a live invitation for the same address keeps exactly one usable
-- link per person.
-- name: DeletePendingInvitation :execrows
DELETE FROM invitations
WHERE organization_id = $1
  AND email_normalized = $2
  AND accepted_at IS NULL;

-- name: CreateInvitation :one
INSERT INTO invitations (
    organization_id,
    email,
    email_normalized,
    display_name,
    role,
    token_hash,
    expires_at,
    created_by
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id;

-- name: FindInvitationByTokenHash :one
SELECT id,
       organization_id,
       email,
       email_normalized,
       display_name,
       role,
       expires_at,
       accepted_at
FROM invitations
WHERE token_hash = $1;

-- name: MarkInvitationAccepted :execrows
UPDATE invitations
SET accepted_at = now(),
    accepted_user_id = $2
WHERE id = $1
  AND accepted_at IS NULL;

-- name: FindInvitationByID :one
SELECT id, organization_id, email, email_normalized, display_name, role, expires_at, accepted_at
FROM invitations
WHERE id = $1;

-- name: RevokeInvitation :execrows
DELETE FROM invitations
WHERE id = $1
  AND organization_id = $2
  AND accepted_at IS NULL;

-- name: HasLocalAccount :one
SELECT EXISTS (
    SELECT 1
    FROM identities
    WHERE organization_id = $1
      AND provider = 'local'
      AND email_normalized = $2
);

-- name: ChangeMembershipRole :execrows
UPDATE memberships
SET role = $3
WHERE organization_id = $1
  AND user_id = $2;
