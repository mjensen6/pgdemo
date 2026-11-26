-- Foreign keys with no index on the referencing columns.
--
-- Postgres indexes the referenced side, a primary or unique key, but not
-- the referencing side. Without that index, every DELETE or key UPDATE on
-- the parent scans the whole child table, and a join from parent to child
-- can't use an index. Read-only.
--
-- The CREATE INDEX column is a starting point. CONCURRENTLY builds it
-- without blocking writes.

SELECT c.conrelid::regclass AS table_name,
       c.conname AS foreign_key,
       c.confrelid::regclass AS references,
       pg_size_pretty(pg_relation_size(c.conrelid)) AS table_size,
       format('CREATE INDEX CONCURRENTLY ON %s (%s);', c.conrelid::regclass,
              (SELECT string_agg(quote_ident(a.attname), ', ' ORDER BY k.n)
               FROM unnest(c.conkey) WITH ORDINALITY k(attnum, n)
               JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = k.attnum)) AS fix
FROM pg_constraint c
WHERE c.contype = 'f'
  -- No index whose leading columns are the foreign key's columns.
  AND NOT EXISTS (
    SELECT FROM pg_index i
    WHERE i.indrelid = c.conrelid
      AND (i.indkey::int2[])[0:cardinality(c.conkey) - 1] @> c.conkey
      AND (i.indkey::int2[])[0:cardinality(c.conkey) - 1] <@ c.conkey
  )
ORDER BY pg_relation_size(c.conrelid) DESC;
