SELECT cron.unschedule('refresh-recent-likes');

DROP MATERIALIZED VIEW IF EXISTS candidate_likes_recent;
