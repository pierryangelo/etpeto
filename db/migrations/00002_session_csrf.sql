-- +goose Up

ALTER TABLE sessions ADD COLUMN csrf_hash bytea;

-- Sessions created before this migration carry no CSRF binding and therefore
-- cannot authorize a mutation. Revoking them forces a new sign-in instead of
-- leaving a session that would silently fail every mutating request.
UPDATE sessions
SET revoked_at = now()
WHERE revoked_at IS NULL
  AND csrf_hash IS NULL;

ALTER TABLE sessions ADD CONSTRAINT sessions_csrf_hash_length
    CHECK (csrf_hash IS NULL OR octet_length(csrf_hash) = 32);

-- +goose Down

ALTER TABLE sessions DROP CONSTRAINT sessions_csrf_hash_length;
ALTER TABLE sessions DROP COLUMN csrf_hash;
