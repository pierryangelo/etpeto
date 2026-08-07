-- name: ListDocuments :many
SELECT id, title, current_revision, updated_at
FROM documents
WHERE organization_id = $1
ORDER BY updated_at DESC
LIMIT 200;

-- name: CreateDocument :one
INSERT INTO documents (
    organization_id,
    created_by,
    document_type,
    title,
    template_revision_id
)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, title, current_revision, updated_at;

-- name: CreateDocumentRevision :one
INSERT INTO document_revisions (
    document_id,
    revision,
    schema_version,
    content,
    content_hash,
    created_by
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, document_id, revision, created_at;

-- name: FindDocument :one
SELECT documents.id,
       documents.title,
       documents.current_revision,
       documents.updated_at,
       document_revisions.content
FROM documents
JOIN document_revisions
  ON document_revisions.document_id = documents.id
 AND document_revisions.revision = documents.current_revision
WHERE documents.id = $1
  AND documents.organization_id = $2;

-- name: DocumentExists :one
SELECT EXISTS (
    SELECT 1
    FROM documents
    WHERE id = $1
      AND organization_id = $2
);

-- name: AdvanceDocumentRevision :one
UPDATE documents
SET current_revision = current_revision + 1,
    title = $3,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
  AND current_revision = $4
RETURNING id, title, current_revision, updated_at;
