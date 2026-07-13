CREATE INDEX CONCURRENTLY candidate_likes_actor_did_created_at_idx ON candidate_likes (
    actor_did, created_at DESC
) WHERE deleted_at IS NULL;
