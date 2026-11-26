-- Indexes that no query has used since statistics were last reset.
--
-- Every index slows down writes and takes disk, so one that's never read
-- is pure cost. Before dropping one, check:
--   - since when: the stats_reset line below. A week of stats can miss a
--     monthly report.
--   - replicas: each server counts its own scans, so check every replica.
-- Unique and primary key indexes are left out: they enforce rules even if
-- no query reads them. Read-only.

SELECT coalesce(stats_reset::text, 'never reset: since the stats began') AS counting_since
FROM pg_stat_database WHERE datname = current_database();

SELECT s.schemaname AS schema, s.relname AS table_name, s.indexrelname AS index_name,
       pg_size_pretty(pg_relation_size(s.indexrelid)) AS size,
       s.idx_scan AS scans,
       s.last_idx_scan AS last_used  -- Postgres 16 and later
FROM pg_stat_user_indexes s
JOIN pg_index i ON i.indexrelid = s.indexrelid
WHERE s.idx_scan = 0
  AND NOT i.indisunique
  AND NOT i.indisprimary
ORDER BY pg_relation_size(s.indexrelid) DESC;
