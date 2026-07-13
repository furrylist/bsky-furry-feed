-- Index to make the materialized view refresh fast
CREATE INDEX CONCURRENTLY candidate_likes_created_at_idx ON candidate_likes (created_at DESC) WHERE deleted_at IS NULL;
