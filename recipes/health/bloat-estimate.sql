-- How much of each table is dead rows and free space: bloat.
--
-- Uses pgstattuple_approx from the pgstattuple extension, which ships with
-- Postgres. It reads only the pages the visibility map can't vouch for, so
-- it's much cheaper than a full scan, but it still reads disk on a big
-- table: run it off-peak. Tables under 1 MB are skipped.
--
-- Free space is normal after VACUUM, which reuses it. A table that stays
-- mostly free space for days is the one worth shrinking, with REPACK or
-- pg_repack.
--
-- Creates the pgstattuple extension. Needs superuser, or the pg_stat_scan_tables role.

CREATE EXTENSION IF NOT EXISTS pgstattuple;

SELECT n.nspname AS schema, c.relname AS table_name,
       pg_size_pretty(s.table_len) AS size,
       round(s.dead_tuple_percent::numeric, 1) AS dead_percent,
       round(s.approx_free_percent::numeric, 1) AS free_percent,
       pg_size_pretty((s.dead_tuple_len + s.approx_free_space)::bigint) AS reclaimable
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
CROSS JOIN LATERAL pgstattuple_approx(c.oid) s
WHERE c.relkind = 'r'
  AND n.nspname NOT IN ('pg_catalog', 'information_schema')
  AND pg_relation_size(c.oid) > 1024 * 1024
ORDER BY s.dead_tuple_len + s.approx_free_space DESC;
