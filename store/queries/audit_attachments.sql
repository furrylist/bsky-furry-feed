-- name: SaveAttachment :one
INSERT INTO audit_attachments (actor_did, mime_type, data)
VALUES ($1, $2, $3)
RETURNING id;

-- name: GetAttachment :one
SELECT
    data,
    mime_type
FROM audit_attachments
WHERE id = $1;
