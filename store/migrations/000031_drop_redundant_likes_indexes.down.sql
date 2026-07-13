CREATE INDEX CONCURRENTLY IF NOT EXISTS candidate_likes_actor_did_idx ON candidate_likes (actor_did);
DROP INDEX CONCURRENTLY IF EXISTS candidate_likes_subject_actor_idx;
