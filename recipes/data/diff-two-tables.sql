-- Diff two tables: which rows were added, removed, or changed.
--
-- Use it to check a migration, a backfill, or a restored copy against the
-- original. Both tables need the same key. Read-only.
--
-- Change: customers_old, customers_new, and id to your tables and key.
-- To see only some columns, compare (a.email, a.plan) instead of a and b.

SELECT coalesce(a.id, b.id) AS id,
       CASE WHEN b.id IS NULL THEN 'removed'
            WHEN a.id IS NULL THEN 'added'
            ELSE 'changed' END AS change,
       to_jsonb(a) AS old_row,
       to_jsonb(b) AS new_row
FROM customers_old a
FULL JOIN customers_new b ON b.id = a.id
-- Comparing the rows as JSON works even when the tables have different
-- columns. Stripping nulls makes a column that's missing on one side equal
-- to NULL there. IS DISTINCT FROM treats two NULLs as equal, where <> would
-- hide any row with a NULL in it. If both tables have exactly the same
-- columns, WHERE a IS DISTINCT FROM b does the same, faster.
WHERE jsonb_strip_nulls(to_jsonb(a)) IS DISTINCT FROM jsonb_strip_nulls(to_jsonb(b))
ORDER BY 1;
