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
            -- Match at least one of the queried hashtags.
            -- If unspecified, do not filter.
            (
                COALESCE(sqlc.narg(hashtags)::TEXT [], '{}') = '{}'
                OR sqlc.arg(hashtags)::TEXT [] && cp.hashtags
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
WITH my_network AS (
    SELECT subject_did AS did
    FROM candidate_follows
    WHERE
        actor_did = sqlc.arg('actor_did')
        AND subject_did != 'did:plc:jdkvwye2lf4mingzk7qdebzc'
    UNION
    SELECT actor_did AS did
    FROM candidate_follows
    WHERE
        subject_did = sqlc.arg('actor_did')
        AND actor_did != 'did:plc:jdkvwye2lf4mingzk7qdebzc'
),

my_network_quiet AS (
    SELECT
        did,
        recent_posts
    FROM (
        SELECT
            did,
            COUNT(*) AS recent_posts
        FROM my_network
        INNER JOIN candidate_posts ON actor_did = did
        WHERE created_at > NOW() - INTERVAL '7 days'
        GROUP BY did
    ) WHERE recent_posts <= 2
),

my_network_yappers AS (
    SELECT
        did,
        recent_posts
    FROM (
        SELECT
            did,
            COUNT(*) AS recent_posts
        FROM my_network
        INNER JOIN candidate_posts ON actor_did = did
        WHERE created_at > NOW() - INTERVAL '7 days'
        GROUP BY did
    ) WHERE recent_posts >= 5
),

my_top_liked_dids AS (
    SELECT
        cp.actor_did AS did,
        COUNT(*) AS liked_count
    FROM candidate_likes AS cl
    INNER JOIN candidate_posts AS cp ON cl.subject_uri = cp.uri
    WHERE
        cl.actor_did = sqlc.arg('actor_did')
        AND cp.created_at > NOW() - INTERVAL '30 days'
    GROUP BY did
    ORDER BY liked_count DESC
    LIMIT 20
),

most_recent_likes_from_my_network AS (
    SELECT cp.actor_did AS did
    FROM candidate_likes AS cl
    INNER JOIN candidate_posts AS cp ON cl.subject_uri = cp.uri
    WHERE cl.actor_did = ANY(SELECT did FROM my_network)
    ORDER BY cp.created_at DESC
    LIMIT 100
),

horniness_rate AS (
    SELECT SUM(is_nsfw::INT) / COUNT(*)::FLOAT AS hornyness_rate
    FROM (
        SELECT
            (ARRAY['nsfw', 'mursuit', 'murrsuit', 'nsfwfurry', 'furrynsfw'] && cp.hashtags)
            OR (ARRAY['porn', 'nudity', 'sexual'] && cp.self_labels) AS is_nsfw
        FROM candidate_likes AS cl
        INNER JOIN candidate_posts AS cp
            ON
                cl.subject_uri = cp.uri
                AND cl.actor_did = sqlc.arg('actor_did')
                AND cp.has_media
                AND cp.created_at > NOW() - INTERVAL '30 days'
    )
)

SELECT
    cp.uri,
    ph.score,
    (
        (ARRAY['nsfw', 'mursuit', 'murrsuit', 'nsfwfurry', 'furrynsfw'] && cp.hashtags)
        OR (ARRAY['porn', 'nudity', 'sexual'] && cp.self_labels)
    ) AS is_nsfw,
    (
        SELECT COUNT(*) + 1
        FROM candidate_likes AS cl2
        WHERE
            subject_uri = cp.uri
            AND (
                cl2.actor_did = ANY(SELECT did FROM my_network)
                AND cl2.actor_did != sqlc.arg('actor_did')
            )
    )
    * (CASE cp.actor_did = ANY(SELECT did FROM most_recent_likes_from_my_network) WHEN TRUE THEN 1.25 ELSE 1 END)
    * (CASE cp.actor_did = ANY(SELECT did FROM my_top_liked_dids) WHEN TRUE THEN 1.75 ELSE 0.5 END)
    * (CASE cp.actor_did = ANY(SELECT did FROM my_network_quiet) WHEN TRUE THEN 2.33 ELSE 1 END)
    * (CASE cp.actor_did = ANY(SELECT did FROM my_network_yappers) WHEN TRUE THEN 0.75 ELSE 1 END)
    * (CASE (
        (ARRAY['nsfw', 'mursuit', 'murrsuit', 'nsfwfurry', 'furrynsfw'] && cp.hashtags)
        OR (ARRAY['porn', 'nudity', 'sexual'] && cp.self_labels)
    ) WHEN TRUE THEN 1 + (SELECT * FROM horniness_rate) * 1.5 ELSE 1
    END)
    * (CASE cp.actor_did = ANY(SELECT did FROM my_network) WHEN TRUE THEN 100 ELSE 0.5 END) AS fluff_relevance_score
FROM candidate_posts AS cp
INNER JOIN candidate_actors AS ca ON cp.actor_did = ca.did
INNER JOIN post_scores AS ph
    ON
        cp.uri = ph.uri AND ph.alg = sqlc.arg(alg)
        AND ph.generation_seq = sqlc.arg(generation_seq)
WHERE
    cp.created_at > NOW() - INTERVAL '7 day'
    AND actor_did != sqlc.arg('actor_did')
    AND (
        actor_did = ANY(SELECT did FROM most_recent_likes_from_my_network)
        OR actor_did = ANY(SELECT did FROM my_network)
        OR actor_did = ANY(SELECT did FROM my_top_liked_dids)
    )
    AND NOT EXISTS (
        SELECT 1
        FROM candidate_likes AS cl
        WHERE
            cl.subject_uri = cp.uri
            AND cl.actor_did = sqlc.arg('actor_did')
            AND cl.created_at > NOW() - INTERVAL '1 day'
    )

    AND (
        COALESCE(sqlc.narg(disallowed_hashtags)::TEXT [], '{}') = '{}'
        OR NOT sqlc.narg(disallowed_hashtags)::TEXT [] && cp.hashtags
    )
ORDER BY
    fluff_relevance_score DESC, ph.score DESC, ph.uri DESC
LIMIT sqlc.arg(_limit);
