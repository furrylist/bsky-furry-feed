-- name: CreateCandidatePost :exec
INSERT INTO
candidate_posts (
    uri,
    actor_did,
    created_at,
    indexed_at,
    hashtags,
    has_media,
    has_video,
    raw,
    self_labels
)
VALUES
($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: SoftDeleteCandidatePost :exec
UPDATE
candidate_posts
SET
    deleted_at = NOW()
WHERE
    uri = $1;

-- name: GetFurryNewFeed :many
WITH args AS (
    SELECT sqlc.narg(allowed_embeds)::TEXT [] AS allowed_embeds
)

SELECT cp.*
FROM
    candidate_posts AS cp
INNER JOIN candidate_actors AS ca ON cp.actor_did = ca.did
NATURAL JOIN args
WHERE
    -- Only include posts by approved actors
    ca.status = 'approved'
    -- Remove posts hidden by our moderators
    AND cp.is_hidden = FALSE
    -- Remove posts deleted by the actors
    AND cp.deleted_at IS NULL
    AND (
    -- Standard criteria.
        (
            -- Match at least one of the queried hashtags, or the text query.
            -- If unspecified, do not filter.
            (
                (
                    COALESCE(sqlc.narg(hashtags)::TEXT [], '{}') = '{}'
                    OR sqlc.arg(hashtags)::TEXT [] && cp.hashtags
                )
                -- Allow a post to contain a text
                OR (
                    sqlc.narg(text_contains)::TEXT IS NOT NULL
                    AND cp.raw ->> 'text' ILIKE sqlc.narg(text_contains)::TEXT
                )
            )
            -- If any hashtags are disallowed, filter them out.
            AND (
                COALESCE(sqlc.narg(disallowed_hashtags)::TEXT [], '{}') = '{}'
                OR NOT sqlc.narg(disallowed_hashtags)::TEXT [] && cp.hashtags
            )
            AND (
                CARDINALITY(args.allowed_embeds) = 0
                OR (
                    'none' = ANY(args.allowed_embeds)
                    AND COALESCE(cp.has_media, FALSE) = FALSE
                    AND COALESCE(cp.has_video, FALSE) = FALSE
                )
                OR (
                    'image' = ANY(args.allowed_embeds)
                    AND COALESCE(cp.has_media, FALSE) = TRUE
                )
                OR (
                    'video' = ANY(args.allowed_embeds)
                    AND COALESCE(cp.has_video, FALSE) = TRUE
                )
            )
            -- Filter by NSFW status. If unspecified, do not filter.
            AND (
                sqlc.narg(is_nsfw)::BOOLEAN IS NULL
                OR (
                    (ARRAY['nsfw', 'mursuit', 'murrsuit', 'nsfwfurry', 'furrynsfw'] && cp.hashtags)
                    OR (ARRAY['porn', 'nudity', 'sexual'] && cp.self_labels)
                ) = sqlc.narg(is_nsfw)
            )
        )
        -- Pinned DID criteria.
        OR cp.actor_did = ANY(sqlc.arg(pinned_dids)::TEXT [])
    )
    -- Remove posts newer than the cursor timestamp
    AND (cp.indexed_at < sqlc.arg(cursor_timestamp))
    AND cp.indexed_at > NOW() - INTERVAL '7 day'
    AND cp.created_at > NOW() - INTERVAL '7 day'
ORDER BY
    cp.indexed_at DESC
LIMIT sqlc.arg(_limit);

-- name: GetPostByURI :one
SELECT *
FROM
    candidate_posts AS cp
WHERE
    cp.uri = sqlc.arg(uri)
LIMIT 1;

-- name: ListScoredPosts :many
WITH args AS (
    SELECT sqlc.narg(allowed_embeds)::TEXT [] AS allowed_embeds
)

SELECT
    cp.*,
    ph.score
FROM
    candidate_posts AS cp
INNER JOIN candidate_actors AS ca ON cp.actor_did = ca.did
INNER JOIN post_scores AS ph
    ON
        cp.uri = ph.uri AND ph.alg = sqlc.arg(alg)
        AND ph.generation_seq = sqlc.arg(generation_seq)
NATURAL JOIN args
WHERE
    cp.is_hidden = FALSE
    AND ca.status = 'approved'
    -- Match at least one of the queried hashtags.
    -- If unspecified, do not filter.
    AND (
        COALESCE(sqlc.narg(hashtags)::TEXT [], '{}') = '{}'
        OR sqlc.narg(hashtags)::TEXT [] && cp.hashtags
    )
    -- If any hashtags are disallowed, filter them out.
    AND (
        COALESCE(sqlc.narg(disallowed_hashtags)::TEXT [], '{}') = '{}'
        OR NOT sqlc.narg(disallowed_hashtags)::TEXT [] && cp.hashtags
    )
    AND (
        CARDINALITY(args.allowed_embeds) = 0
        OR (
            'none' = ANY(args.allowed_embeds)
            AND COALESCE(cp.has_media, FALSE) = FALSE
            AND COALESCE(cp.has_video, FALSE) = FALSE
        )
        OR (
            'image' = ANY(args.allowed_embeds)
            AND COALESCE(cp.has_media, FALSE) = TRUE
        )
        OR (
            'video' = ANY(args.allowed_embeds)
            AND COALESCE(cp.has_video, FALSE) = TRUE
        )
    )
    -- Filter by NSFW status. If unspecified, do not filter.
    AND (
        sqlc.narg(is_nsfw)::BOOLEAN IS NULL
        OR (
            (ARRAY['nsfw', 'mursuit', 'murrsuit', 'nsfwfurry', 'furrynsfw'] && cp.hashtags)
            OR (ARRAY['porn', 'nudity', 'sexual'] && cp.self_labels)
        ) = sqlc.narg(is_nsfw)
    )
    AND cp.deleted_at IS NULL
    AND (
        ROW(ph.score, ph.uri)
        < ROW((sqlc.arg(after_score))::REAL, (sqlc.arg(after_uri))::TEXT)
    )
    AND cp.indexed_at > NOW() - INTERVAL '7 day'
    AND cp.created_at > NOW() - INTERVAL '7 day'
ORDER BY
    ph.score DESC, ph.uri DESC
LIMIT sqlc.arg(_limit);

-- name: ListTestFeedPosts :many
WITH my_recent_likes AS (
    SELECT cl.subject_uri
    FROM candidate_likes AS cl
    WHERE
        cl.actor_did = sqlc.arg('actor_did')
        AND cl.deleted_at IS NULL
        AND cl.created_at > NOW() - INTERVAL '30 days'
    LIMIT 500
),

similar_users AS (
    SELECT
        cl.actor_did AS did,
        COUNT(*) AS shared_likes
    FROM candidate_likes AS cl
    WHERE
        cl.subject_uri IN (SELECT subject_uri FROM my_recent_likes)
        AND cl.actor_did != sqlc.arg('actor_did')
        AND cl.deleted_at IS NULL
    GROUP BY cl.actor_did
    HAVING COUNT(*) >= 2
    ORDER BY shared_likes DESC
    LIMIT 100
),

candidate_posts_recent AS (
    SELECT
        cp.uri,
        cp.actor_did,
        cp.created_at,
        cp.hashtags,
        cp.self_labels
    FROM candidate_posts AS cp
    INNER JOIN candidate_actors AS ca ON cp.actor_did = ca.did
    WHERE
        cp.deleted_at IS NULL
        AND ca.status = 'approved'
        AND cp.created_at > NOW() - INTERVAL '3 days'
        AND (
            COALESCE(sqlc.narg(disallowed_hashtags)::TEXT [], '{}') = '{}'
            OR NOT sqlc.narg(disallowed_hashtags)::TEXT [] && cp.hashtags
        )
),

nsfw_engagement AS (
    SELECT
        COUNT(*) FILTER (
            WHERE
            (ARRAY['nsfw', 'mursuit', 'murrsuit', 'nsfwfurry', 'furrynsfw'] && cp.hashtags)
            OR (ARRAY['porn', 'nudity', 'sexual'] && cp.self_labels)
        )::FLOAT / GREATEST(COUNT(*), 1) AS nsfw_ratio
    FROM candidate_likes AS cl
    INNER JOIN candidate_posts AS cp ON cl.subject_uri = cp.uri
    WHERE
        cl.actor_did = sqlc.arg('actor_did')
        AND cl.deleted_at IS NULL
        AND cl.created_at > NOW() - INTERVAL '30 days'
),

my_follows AS (
    SELECT subject_did
    FROM candidate_follows
    WHERE
        actor_did = sqlc.arg('actor_did')
        AND deleted_at IS NULL
),

liked_authors AS (
    SELECT DISTINCT cp.actor_did
    FROM candidate_likes AS cl
    INNER JOIN candidate_posts AS cp ON cl.subject_uri = cp.uri
    WHERE
        cl.actor_did = sqlc.arg('actor_did')
        AND cl.deleted_at IS NULL
        AND cl.created_at > NOW() - INTERVAL '30 days'
),

their_recent_likes AS (
    SELECT
        cl.subject_uri,
        COUNT(*) AS like_count,
        MAX(cl.created_at) AS most_recent_like_at,
        BOOL_OR(su.did IN (SELECT subject_did FROM my_follows)) AS liked_by_friend
    FROM candidate_likes_recent AS cl
    INNER JOIN similar_users AS su ON cl.actor_did = su.did
    INNER JOIN candidate_posts_recent AS cpr ON cl.subject_uri = cpr.uri
    WHERE
        NOT EXISTS (
            SELECT 1 FROM candidate_likes_recent AS my_like
            WHERE
                my_like.subject_uri = cl.subject_uri
                AND my_like.actor_did = sqlc.arg('actor_did')
        )
    GROUP BY cl.subject_uri
    HAVING COUNT(*) >= 1
),

scored_candidates AS MATERIALIZED (
    SELECT
        cpr.uri,
        cpr.actor_did,
        trl.like_count,
        trl.most_recent_like_at,
        -- This is all VERY vibes based. There ARE better values because
        -- this gives us a lot of recently posted posts.
        -- I added some random time jitter, so we have more variance. This,
        -- however, might have to be added to the cursor?
        CASE
            WHEN cpr.actor_did IN (SELECT subject_did FROM my_follows)
                THEN trl.most_recent_like_at + INTERVAL '2 hours'
            WHEN cpr.actor_did IN (SELECT actor_did FROM liked_authors)
                THEN trl.most_recent_like_at + INTERVAL '1 hour'
            WHEN trl.liked_by_friend
                THEN trl.most_recent_like_at + INTERVAL '30 minutes'
            WHEN trl.like_count < 5
                THEN trl.most_recent_like_at + INTERVAL '10 minutes'
            ELSE trl.most_recent_like_at - INTERVAL '2 hours'
        END
        -- Penalize NSFW posts for users who don't engage much with NSFW content.
        -- nsfw_ratio < 10% means low engagement. Debuff scales linearly to -6 hours.
        - CASE
            WHEN (
                (ARRAY['nsfw', 'mursuit', 'murrsuit', 'nsfwfurry', 'furrynsfw'] && cpr.hashtags)
                OR (ARRAY['porn', 'nudity', 'sexual'] && cpr.self_labels)
            )
                THEN (1.0 - LEAST((SELECT nsfw_ratio FROM nsfw_engagement) / 0.1, 1.0)) * INTERVAL '6 hours'
            ELSE INTERVAL '0'
        END
        + (RANDOM() - 0.25) * INTERVAL '1 hour' AS boosted_time
    FROM their_recent_likes AS trl
    INNER JOIN candidate_posts_recent AS cpr ON trl.subject_uri = cpr.uri
)

SELECT
    sc.uri,
    sc.actor_did,
    EXTRACT(EPOCH FROM sc.boosted_time)::FLOAT AS fluff_relevance_score
FROM scored_candidates AS sc
WHERE
    ROW(sc.boosted_time, sc.uri)
    < ROW(TO_TIMESTAMP((sqlc.arg(after_score))::DOUBLE PRECISION), (sqlc.arg(after_uri))::TEXT)
ORDER BY sc.boosted_time DESC, sc.uri DESC
LIMIT sqlc.arg(_limit);
