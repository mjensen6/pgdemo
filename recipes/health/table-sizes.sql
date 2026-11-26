-- The biggest tables, split into table data, indexes, and TOAST.
--
-- TOAST holds large values, such as long text and JSON, stored apart from
-- the row. Read-only, and fast: it reads sizes, not rows.

SELECT n.nspname AS schema, c.relname AS table_name,
       pg_size_pretty(pg_total_relation_size(c.oid)) AS total,
       pg_size_pretty(pg_relation_size(c.oid)) AS data,
       pg_size_pretty(pg_indexes_size(c.oid)) AS indexes,
       pg_size_pretty(coalesce(pg_total_relation_size(nullif(c.reltoastrelid, 0)), 0)) AS toast,
       nullif(c.reltuples, -1)::bigint AS estimated_rows  -- blank: never analyzed
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind IN ('r', 'm')
  AND n.nspname NOT IN ('pg_catalog', 'information_schema')
ORDER BY pg_total_relation_size(c.oid) DESC
LIMIT 20;
