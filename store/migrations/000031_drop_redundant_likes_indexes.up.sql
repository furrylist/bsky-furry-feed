-- The composite index candidate_likes_actor_did_created_at_idx covers candidate_likes_actor_did_idx
DROP INDEX IF EXISTS candidate_likes_actor_did_idx;

-- Create composite index for the "already liked" check to replace candidate_likes_subject_uri_idx
CREATE INDEX IF NOT EXISTS candidate_likes_subject_actor_idx ON candidate_likes (
    subject_uri, actor_did
) WHERE deleted_at IS NULL;
