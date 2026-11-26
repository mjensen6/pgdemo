-- Diff two tables with no shared key: rows in one and not the other.
--
-- The quickest diff, and it needs no key, but a changed row shows twice:
-- once as its old version and once as its new one. Both queries must list
-- the same columns in the same order. Duplicate rows count once; use
-- EXCEPT ALL to count every copy. Read-only.
--
-- Change: customers_old, customers_new, and the column list.

SELECT 'only in old' AS side, * FROM (
  SELECT id, email, name, plan FROM customers_old
  EXCEPT
  SELECT id, email, name, plan FROM customers_new
) o
UNION ALL
SELECT 'only in new', * FROM (
  SELECT id, email, name, plan FROM customers_new
  EXCEPT
  SELECT id, email, name, plan FROM customers_old
) n
ORDER BY id, side;
