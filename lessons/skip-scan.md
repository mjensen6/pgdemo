---
title: Skip scan, new in Postgres 18
tldr: A multicolumn index now helps queries that leave out its first column, by hopping between that column's values.
---

```setup
CREATE TABLE sales AS
  SELECT i AS id, i % 4 AS region, i % 5000 AS product FROM generate_series(1, 200000) i;
CREATE INDEX sales_region_product ON sales (region, product);
ANALYZE sales;
CREATE TABLE visits AS
  SELECT i AS id, (ARRAY['DE', 'FR', 'US'])[i % 3 + 1] AS country,
         date '2026-01-01' + i % 365 AS day
  FROM generate_series(1, 200000) i;
ANALYZE visits;
```

```view
SELECT c.relname AS index, pg_get_indexdef(i.indexrelid) AS definition, s.idx_scan AS times_used
FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid
JOIN pg_stat_user_indexes s ON s.indexrelid = i.indexrelid
WHERE i.indrelid IN ('sales'::regclass, 'visits'::regclass) ORDER BY 1;
```

A **multicolumn index** on `(region, product)` is sorted by `region` first, then by `product` within each region, like a phone book sorted by last name, then first name.

That order makes it great for `WHERE region = 2 AND product = 42`, and for `WHERE region = 2` alone. But a query on `product` alone used to get little help from it. The products are spread across every region's part of the index, so Postgres had to read the whole index or the whole table. It's like finding everyone named "Ada" in a phone book.

**Skip scan** fixes that. Postgres jumps to the first region, looks up product 42 there, skips to the next region, looks it up again, and so on. With 4 regions, that's a handful of quick lookups instead of reading everything.

**When it helps.** Skip scan works best when the leading column has **few distinct values**, like region, status, or country. Then there are only a few places to hop to. If the leading column has millions of distinct values, there are millions of hops, and Postgres sensibly chooses another plan. So for a column queried on its own, put the column with few values first. Then one index can serve both kinds of query.

**Where it came from.** Skip scan arrived in Postgres 18 (2025), written by Peter Geoghegan. Oracle has had index skip scans since 2001. Before 18, the workaround was a recursive query that hopped between values by hand, or a second index.

**The Live View** shows the indexes on `sales` and `visits`:

| Column | Meaning |
|---|---|
| `index` | The index name |
| `definition` | Its columns, in order |
| `times_used` | How many times a query has used it |

## Try it

Count sales of product 42, a query that never mentions `region`. The plan uses the `(region, product)` index anyway, and `Index Searches: 5` is the skip scan: one hop per region, plus one to find there are no more. In the Live View, `times_used` goes up.

```sql
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF, BUFFERS OFF)
SELECT count(*) FROM sales WHERE product = 42;
```

```expect
sales_region_product
Index Searches: 5
```

Give it both columns, and it's an ordinary lookup: `Index Searches: 1`.

```sql
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF, BUFFERS OFF)
SELECT count(*) FROM sales WHERE region = 2 AND product = 42;
```

```expect
Index Searches: 1
```

Skip scan works because `region` has only 4 distinct values. `n_distinct` is the planner's estimate of how many different values each column has.

```sql
SELECT attname AS column, n_distinct FROM pg_stats WHERE tablename = 'sales' ORDER BY attname;
```

```expect
region  |          4
```

## Challenge

Table `visits` has no index yet. Queries filter by `country` (3 values) or by `day` (365 values), and sometimes both. Create one index that skip scan can use for a query on `day` alone.

Hint: Which column has fewer distinct values? Put it first, like region in sales.

```solution
CREATE INDEX visits_country_day ON visits (country, day);
```

```check
-- An index on visits that starts with country, then day.
SELECT EXISTS (SELECT FROM pg_index
               WHERE indrelid = 'visits'::regclass
                 AND indkey[0] = (SELECT attnum FROM pg_attribute WHERE attrelid = 'visits'::regclass AND attname = 'country')
                 AND indkey[1] = (SELECT attnum FROM pg_attribute WHERE attrelid = 'visits'::regclass AND attname = 'day'));
```
