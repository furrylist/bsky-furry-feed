DROP INDEX IF EXISTS candidate_posts_indexed_at_idx;

CREATE INDEX candidate_posts_indexed_at_idx ON public.candidate_posts (indexed_at, is_hidden);
CREATE INDEX candidate_posts_created_at_idx ON public.candidate_posts (created_at);
