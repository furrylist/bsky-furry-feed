CREATE MATERIALIZED VIEW candidate_likes_recent AS
SELECT
    uri,
    actor_did,
    subject_uri,
    created_at,
    indexed_at
FROM candidate_likes
WHERE
    deleted_at IS NULL
    AND created_at > NOW() - INTERVAL '7 days';

CREATE UNIQUE INDEX candidate_likes_recent_uri_idx ON candidate_likes_recent (uri);
CREATE INDEX candidate_likes_recent_subject_actor_idx ON candidate_likes_recent (subject_uri, actor_did);
CREATE INDEX candidate_likes_recent_actor_created_idx ON candidate_likes_recent (actor_did, created_at DESC);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_cron') THEN
        PERFORM cron.schedule(
            'refresh-recent-likes',
            '0 * * * *',
            'REFRESH MATERIALIZED VIEW CONCURRENTLY candidate_likes_recent;'
        );
    END IF;
END $$;
