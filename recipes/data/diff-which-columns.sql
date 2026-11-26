-- Diff two tables column by column: for each changed row, list only the
-- columns that differ, with the old and new values.
--
-- Works for any tables with a shared key, whatever their columns, because
-- it compares the rows as JSON. A column that exists in only one table
-- shows as changed from or to null. Read-only.
--
-- Change: customers_old, customers_new, and id to your tables and key.

SELECT a.id, col.key AS column_name,
       to_jsonb(a) -> col.key AS old_value,
       to_jsonb(b) -> col.key AS new_value
FROM customers_old a
JOIN customers_new b ON b.id = a.id
CROSS JOIN LATERAL (
  SELECT key FROM jsonb_object_keys(to_jsonb(a) || to_jsonb(b)) AS k(key)
) col
-- A column missing on one side counts as null there, so it shows only
-- when the other side has a value.
WHERE coalesce(to_jsonb(a) -> col.key, 'null') <> coalesce(to_jsonb(b) -> col.key, 'null')
ORDER BY a.id, column_name;
