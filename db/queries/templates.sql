-- name: ListTemplates :many
SELECT templates.id,
       templates.document_type,
       templates.name,
       templates.draft_version,
       templates.draft_updated_at,
       (
           SELECT max(revision)
           FROM template_revisions
           WHERE template_revisions.template_id = templates.id
       ) AS published_revision,
       (
           SELECT count(*)
           FROM documents
           JOIN template_revisions ON template_revisions.id = documents.template_revision_id
           WHERE template_revisions.template_id = templates.id
       ) AS document_count
FROM templates
WHERE templates.organization_id = $1
ORDER BY templates.name;

-- name: FindTemplate :one
SELECT id, document_type, name, draft, draft_version, draft_updated_at
FROM templates
WHERE id = $1
  AND organization_id = $2;

-- name: FindTemplateByType :one
SELECT id, document_type, name, draft, draft_version, draft_updated_at
FROM templates
WHERE organization_id = $1
  AND document_type = $2;

-- name: CreateTemplate :one
INSERT INTO templates (organization_id, document_type, name, draft, draft_updated_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, document_type, name, draft, draft_version, draft_updated_at;

-- name: SaveTemplateDraft :one
UPDATE templates
SET draft = $3,
    draft_version = draft_version + 1,
    draft_updated_at = now(),
    draft_updated_by = $4
WHERE id = $1
  AND organization_id = $2
  AND draft_version = $5
RETURNING id, document_type, name, draft, draft_version, draft_updated_at;

-- name: TemplateExists :one
SELECT EXISTS (
    SELECT 1 FROM templates WHERE id = $1 AND organization_id = $2
);

-- name: PublishTemplateRevision :one
INSERT INTO template_revisions (template_id, revision, blueprint, schema_version, published_by)
VALUES (
    $1,
    coalesce(
        (SELECT max(revision) + 1 FROM template_revisions WHERE template_id = $1),
        1
    ),
    $2,
    $3,
    $4
)
RETURNING id, template_id, revision, published_at;

-- name: FindLatestPublishedRevision :one
SELECT template_revisions.id,
       template_revisions.template_id,
       template_revisions.revision,
       template_revisions.blueprint,
       template_revisions.published_at
FROM template_revisions
JOIN templates ON templates.id = template_revisions.template_id
WHERE templates.organization_id = $1
  AND templates.document_type = $2
ORDER BY template_revisions.revision DESC
LIMIT 1;

-- name: FindPublishedRevision :one
SELECT template_revisions.id,
       template_revisions.template_id,
       template_revisions.revision,
       template_revisions.blueprint,
       template_revisions.published_at
FROM template_revisions
JOIN templates ON templates.id = template_revisions.template_id
WHERE template_revisions.id = $1
  AND templates.organization_id = $2;

-- name: ListPublishedRevisions :many
SELECT template_revisions.id,
       template_revisions.revision,
       template_revisions.published_at,
       users.display_name AS published_by_name
FROM template_revisions
JOIN users ON users.id = template_revisions.published_by
WHERE template_revisions.template_id = $1
ORDER BY template_revisions.revision DESC
LIMIT 50;

-- name: ListPublishedTemplates :many
SELECT DISTINCT ON (templates.id)
       template_revisions.id AS revision_id,
       templates.id AS template_id,
       templates.name,
       templates.document_type,
       template_revisions.revision,
       template_revisions.published_at
FROM templates
JOIN template_revisions ON template_revisions.template_id = templates.id
WHERE templates.organization_id = $1
ORDER BY templates.id, template_revisions.revision DESC;
