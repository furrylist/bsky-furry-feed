DROP INDEX IF EXISTS candidate_posts_indexed_at_idx;
DROP INDEX IF EXISTS candidate_posts_created_at_idx;

CREATE INDEX candidate_posts_indexed_at_idx ON public.candidate_posts (
    indexed_at DESC, created_at DESC, is_hidden
)
WHERE deleted_at IS NULL;
