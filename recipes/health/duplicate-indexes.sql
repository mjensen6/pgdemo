-- Indexes that are exact copies of another index on the same table.
--
-- Same columns, same operators, same expression and WHERE clause. Keep one
-- and drop the rest; if one of them backs a constraint, keep that one.
-- Read-only.

SELECT i.indrelid::regclass AS table_name,
       array_agg(i.indexrelid::regclass ORDER BY i.indexrelid) AS same_indexes,
       pg_size_pretty(sum(pg_relation_size(i.indexrelid))::bigint) AS total_size
FROM pg_index i
JOIN pg_class c ON c.oid = i.indrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
GROUP BY i.indrelid, i.indkey::text, i.indclass::text, i.indcollation::text,
         coalesce(pg_get_expr(i.indexprs, i.indrelid), ''),
         coalesce(pg_get_expr(i.indpred, i.indrelid), '')
HAVING count(*) > 1
ORDER BY sum(pg_relation_size(i.indexrelid)) DESC;
