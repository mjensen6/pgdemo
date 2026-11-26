-- Find which part of two big tables differs, by checksumming blocks of
-- rows, without diffing every row.
--
-- Each block of 10,000 ids gets one md5 of all its rows. Only the blocks
-- whose checksums differ need a full diff: run diff-two-tables.sql with
-- WHERE id BETWEEN first_id AND last_id. Reads both tables in full once,
-- so it suits millions of rows, and pairs with diff-across-databases.sql
-- for tables on two servers. Read-only.
--
-- Change: events_a, events_b, id, and the block size.

WITH a AS (
  SELECT id / 10000 AS block, count(*) AS n, md5(string_agg(t::text, '|' ORDER BY id)) AS sum
  FROM events_a t GROUP BY 1
), b AS (
  SELECT id / 10000 AS block, count(*) AS n, md5(string_agg(t::text, '|' ORDER BY id)) AS sum
  FROM events_b t GROUP BY 1
)
SELECT coalesce(a.block, b.block) * 10000 AS first_id,
       coalesce(a.block, b.block) * 10000 + 9999 AS last_id,
       a.n AS rows_a, b.n AS rows_b
FROM a FULL JOIN b ON b.block = a.block
WHERE a.sum IS DISTINCT FROM b.sum
ORDER BY 1;
