-- 04 pg_stat_statements: find your slowest queries
CREATE EXTENSION pg_stat_statements;
CREATE TABLE orders AS SELECT i AS id, i % 100 AS customer FROM generate_series(1, 100000) i;
SELECT pg_stat_statements_reset() IS NOT NULL AS reset;
SELECT count(*) FROM orders WHERE customer = 7;
SELECT count(*) FROM orders WHERE customer = 8;
SELECT count(*) FROM orders WHERE id = 5;
-- Similar queries are grouped, with values replaced by $1.
SELECT calls, round(total_exec_time::numeric, 2) AS total_ms, query
FROM pg_stat_statements WHERE query LIKE 'SELECT count%' ORDER BY total_exec_time DESC;
