-- +goose Up

CREATE TABLE organizations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug text NOT NULL UNIQUE,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT organizations_slug_format CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
    CONSTRAINT organizations_name_not_blank CHECK (btrim(name) <> '')
);

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name text NOT NULL,
    disabled_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_display_name_not_blank CHECK (btrim(display_name) <> '')
);

CREATE TABLE identities (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider text NOT NULL,
    issuer text,
    subject text,
    email text NOT NULL,
    email_normalized text NOT NULL,
    password_hash text,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT identities_provider CHECK (provider IN ('local', 'oidc')),
    CONSTRAINT identities_email_not_blank CHECK (btrim(email_normalized) <> ''),
    CONSTRAINT identities_provider_fields CHECK (
        (
            provider = 'local'
            AND issuer IS NULL
            AND subject IS NULL
            AND password_hash IS NOT NULL
        )
        OR
        (
            provider = 'oidc'
            AND issuer IS NOT NULL
            AND subject IS NOT NULL
            AND password_hash IS NULL
        )
    )
);

CREATE UNIQUE INDEX identities_local_email_unique
    ON identities (organization_id, email_normalized)
    WHERE provider = 'local';

CREATE UNIQUE INDEX identities_oidc_subject_unique
    ON identities (issuer, subject)
    WHERE provider = 'oidc';

CREATE INDEX identities_user_id_idx ON identities (user_id);

CREATE TABLE memberships (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT memberships_unique UNIQUE (organization_id, user_id),
    CONSTRAINT memberships_role CHECK (
        role IN ('installation_admin', 'organization_admin', 'template_manager', 'author')
    )
);

CREATE INDEX memberships_user_id_idx ON memberships (user_id);

CREATE TABLE sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT sessions_token_hash_length CHECK (octet_length(token_hash) = 32),
    CONSTRAINT sessions_expiry_after_creation CHECK (expires_at > created_at)
);

CREATE INDEX sessions_active_user_idx
    ON sessions (user_id, expires_at)
    WHERE revoked_at IS NULL;

-- +goose Down

DROP TABLE sessions;
DROP TABLE memberships;
DROP TABLE identities;
DROP TABLE users;
DROP TABLE organizations;

