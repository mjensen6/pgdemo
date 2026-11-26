-- Tables with no primary key.
--
-- Without one, nothing stops duplicate rows, and some tools can't handle
-- the table at all: logical replication can't send its UPDATEs and
-- DELETEs, and pg_repack refuses to rebuild it. Read-only.

SELECT n.nspname AS schema, c.relname AS table_name,
       nullif(c.reltuples, -1)::bigint AS estimated_rows,  -- blank: never analyzed
       EXISTS (SELECT FROM pg_index i WHERE i.indrelid = c.oid AND i.indisunique) AS has_unique_index
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind IN ('r', 'p')
  AND n.nspname NOT IN ('pg_catalog', 'information_schema')
  AND n.nspname NOT LIKE 'pg_toast%'
  AND NOT EXISTS (SELECT FROM pg_constraint k WHERE k.conrelid = c.oid AND k.contype = 'p')
ORDER BY 1, 2;
