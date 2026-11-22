-- 01 MVCC and bloat: why updates leave dead rows behind
CREATE EXTENSION pgstattuple;
CREATE TABLE t (id int, note text) WITH (autovacuum_enabled = false);
INSERT INTO t SELECT i, 'hello' FROM generate_series(1, 10000) i;
SELECT pg_size_pretty(pg_table_size('t')) AS size, tuple_count, dead_tuple_count FROM pgstattuple('t');

-- An UPDATE writes a new row version and leaves the old one dead.
UPDATE t SET note = 'world';
SELECT pg_size_pretty(pg_table_size('t')) AS size, tuple_count, dead_tuple_count FROM pgstattuple('t');

-- VACUUM clears dead rows for reuse, but the file usually stays big. That is bloat.
VACUUM t;
SELECT pg_size_pretty(pg_table_size('t')) AS size, tuple_count, dead_tuple_count, round(free_percent) AS free_pct FROM pgstattuple('t');
