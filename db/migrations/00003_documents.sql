-- +goose Up

CREATE TABLE documents (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    created_by uuid NOT NULL REFERENCES users (id),
    document_type text NOT NULL,
    title text NOT NULL,
    current_revision integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT documents_title_not_blank CHECK (btrim(title) <> ''),
    CONSTRAINT documents_type CHECK (document_type IN ('etp')),
    CONSTRAINT documents_current_revision_positive CHECK (current_revision > 0)
);

CREATE INDEX documents_organization_recent_idx
    ON documents (organization_id, updated_at DESC);

-- Every save appends a row. Nothing is updated in place, so a working revision
-- can always be recovered and a future issuance can freeze one of them.
CREATE TABLE document_revisions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id uuid NOT NULL REFERENCES documents (id) ON DELETE CASCADE,
    revision integer NOT NULL,
    schema_version integer NOT NULL,
    content jsonb NOT NULL,
    content_hash bytea NOT NULL,
    created_by uuid NOT NULL REFERENCES users (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT document_revisions_unique UNIQUE (document_id, revision),
    CONSTRAINT document_revisions_positive CHECK (revision > 0),
    CONSTRAINT document_revisions_schema_version_positive CHECK (schema_version > 0),
    CONSTRAINT document_revisions_hash_length CHECK (octet_length(content_hash) = 32)
);

CREATE INDEX document_revisions_document_idx
    ON document_revisions (document_id, revision DESC);

-- +goose Down

DROP TABLE document_revisions;
DROP TABLE documents;
