-- Columns that are almost always NULL.
--
-- They may be unused, or a sign that the data belongs in another table.
-- Uses the planner's statistics, so it's instant but only as fresh as the
-- last ANALYZE. Read-only.
--
-- Change: the 0.9 threshold.

SELECT schemaname AS schema, tablename AS table_name, attname AS column_name,
       round(100 * null_frac::numeric, 1) AS null_percent
FROM pg_stats
WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
  AND null_frac > 0.9
ORDER BY null_frac DESC, 1, 2, 3;
