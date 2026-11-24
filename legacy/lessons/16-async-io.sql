-- 16 Async I/O (PG18): read ahead instead of one page at a time
CREATE EXTENSION pg_buffercache;
SHOW io_method;
CREATE TABLE big AS SELECT i, repeat('x', 100) AS pad FROM generate_series(1, 500000) i;
SELECT pg_size_pretty(pg_table_size('big')) AS size;
-- Empty the cache, then scan with read-ahead turned off and on.
\timing on
SET max_parallel_workers_per_gather = 0;
SET effective_io_concurrency = 0;
SELECT buffers_evicted > 0 AS evicted FROM pg_buffercache_evict_relation('big');
SELECT count(*) FROM big;
SET effective_io_concurrency = 64;
SELECT buffers_evicted > 0 AS evicted FROM pg_buffercache_evict_relation('big');
SELECT count(*) FROM big;
\timing off
-- The gap is largest on slow or network disks. Fast SSDs and the OS cache shrink it.
