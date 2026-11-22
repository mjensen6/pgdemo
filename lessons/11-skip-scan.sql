-- 11 Skip scan (PG18): use an index without its first column
CREATE TABLE sales AS SELECT i AS id, i % 4 AS region, i % 5000 AS product FROM generate_series(1, 200000) i;
CREATE INDEX ON sales (region, product);
ANALYZE sales;
-- The query never mentions region, yet the index still works.
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF) SELECT count(*) FROM sales WHERE product = 42;
-- "Index Searches: 5" is the skip scan hopping between the 4 regions.
