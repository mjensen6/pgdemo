-- The queries that use the most total time, from pg_stat_statements.
--
-- Total time finds the queries worth fixing first: a 2 ms query run a
-- million times costs more than a 5 s query run once. Queries that differ
-- only in their values are grouped into one shape.
--
-- Needs pg_stat_statements in shared_preload_libraries, then
-- CREATE EXTENSION pg_stat_statements once per database. Read-only.
-- To start counting afresh: SELECT pg_stat_statements_reset();

SELECT round(total_exec_time::numeric, 1) AS total_ms,
       calls,
       round(mean_exec_time::numeric, 2) AS mean_ms,
       round((100 * total_exec_time / sum(total_exec_time) OVER ())::numeric, 1) AS percent_of_all,
       rows,
       left(regexp_replace(query, '\s+', ' ', 'g'), 100) AS query
FROM pg_stat_statements
WHERE dbid = (SELECT oid FROM pg_database WHERE datname = current_database())
ORDER BY total_exec_time DESC
LIMIT 15;
