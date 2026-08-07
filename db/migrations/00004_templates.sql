-- +goose Up

-- A template is the institutional script for one document type. It always has
-- a draft, which only the manager sees, and zero or more published revisions.
CREATE TABLE templates (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    document_type text NOT NULL,
    name text NOT NULL,
    draft jsonb NOT NULL,
    -- Bumped on every draft write so a stale editor cannot overwrite a newer
    -- draft, the same optimistic locking used for documents.
    draft_version integer NOT NULL DEFAULT 1,
    draft_updated_at timestamptz NOT NULL DEFAULT now(),
    draft_updated_by uuid REFERENCES users (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT templates_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT templates_document_type CHECK (document_type IN ('etp')),
    CONSTRAINT templates_draft_version_positive CHECK (draft_version > 0),
    -- One script per document type per organization keeps "the ETP script"
    -- unambiguous for everyone writing one.
    CONSTRAINT templates_type_unique UNIQUE (organization_id, document_type)
);

-- Publishing appends a row. A published revision is never updated: documents
-- created from it must keep reading exactly what was published.
CREATE TABLE template_revisions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id uuid NOT NULL REFERENCES templates (id) ON DELETE CASCADE,
    revision integer NOT NULL,
    blueprint jsonb NOT NULL,
    schema_version integer NOT NULL,
    published_at timestamptz NOT NULL DEFAULT now(),
    published_by uuid NOT NULL REFERENCES users (id),
    CONSTRAINT template_revisions_unique UNIQUE (template_id, revision),
    CONSTRAINT template_revisions_positive CHECK (revision > 0),
    CONSTRAINT template_revisions_schema_version_positive CHECK (schema_version > 0)
);

CREATE INDEX template_revisions_recent_idx
    ON template_revisions (template_id, revision DESC);

-- A document is bound to the revision it was created from and never migrates
-- on its own. The column is nullable because documents created before this
-- migration predate published scripts.
ALTER TABLE documents
    ADD COLUMN template_revision_id uuid REFERENCES template_revisions (id);

CREATE INDEX documents_template_revision_idx
    ON documents (template_revision_id);

-- +goose Down

DROP INDEX documents_template_revision_idx;
ALTER TABLE documents DROP COLUMN template_revision_id;
DROP TABLE template_revisions;
DROP TABLE templates;
