-- Exact row counts for every table, next to the planner's estimate.
--
-- The estimate is free and usually close. The exact count runs
-- count(*) on each table, which reads them in full, so it can take a while
-- on a big database, and waits behind any lock that blocks reads, such as
-- a running ALTER TABLE. Use it to compare two copies of a database table
-- by table. Read-only.
--
-- Change: the schema list.

SELECT n.nspname AS schema, c.relname AS table_name,
       -- query_to_xml runs a query built from a string, inside a plain SELECT.
       (xpath('/row/n/text()',
              query_to_xml(format('SELECT count(*) AS n FROM %I.%I', n.nspname, c.relname),
                           false, true, '')))[1]::text::bigint AS exact_rows,
       nullif(c.reltuples, -1)::bigint AS estimated_rows  -- blank: never analyzed
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind IN ('r', 'p')
  AND n.nspname NOT IN ('pg_catalog', 'information_schema')
  AND n.nspname NOT LIKE 'pg_toast%'
ORDER BY 1, 2;
