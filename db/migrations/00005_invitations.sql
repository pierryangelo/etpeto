-- +goose Up

-- A single-use invitation. It carries no password: the person sets their own
-- when accepting, so the administration never learns it. Only the hash of the
-- token is stored, exactly as with a session.
CREATE TABLE invitations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    email text NOT NULL,
    email_normalized text NOT NULL,
    display_name text NOT NULL,
    role text NOT NULL,
    token_hash bytea NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    accepted_at timestamptz,
    accepted_user_id uuid REFERENCES users (id),
    created_by uuid NOT NULL REFERENCES users (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT invitations_token_hash_length CHECK (octet_length(token_hash) = 32),
    CONSTRAINT invitations_email_not_blank CHECK (btrim(email_normalized) <> ''),
    CONSTRAINT invitations_display_name_not_blank CHECK (btrim(display_name) <> ''),
    CONSTRAINT invitations_role CHECK (
        role IN ('installation_admin', 'organization_admin', 'template_manager', 'author')
    ),
    -- Accepting records both the moment and who was created, never one alone.
    CONSTRAINT invitations_acceptance_complete CHECK (
        (accepted_at IS NULL AND accepted_user_id IS NULL)
        OR (accepted_at IS NOT NULL AND accepted_user_id IS NOT NULL)
    )
);

-- One live invitation per address. Inviting the same person twice replaces the
-- previous link instead of leaving two usable ones.
CREATE UNIQUE INDEX invitations_pending_email_unique
    ON invitations (organization_id, email_normalized)
    WHERE accepted_at IS NULL;

CREATE INDEX invitations_organization_idx ON invitations (organization_id);

-- +goose Down

DROP TABLE invitations;
