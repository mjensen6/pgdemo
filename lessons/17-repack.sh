# 17 REPACK (PG19): shrink a table while it stays online
. "$(dirname "$0")/../lib/helpers.sh"
quiet "CREATE TABLE t (id int PRIMARY KEY, n int DEFAULT 0, pad text) WITH (autovacuum_enabled = false);
       INSERT INTO t SELECT i, 0, repeat('x', 100) FROM generate_series(1, 200000) i;
       DELETE FROM t WHERE id % 10 <> 0;"
quiet "VACUUM t"
sql "SELECT pg_size_pretty(pg_table_size('t')) AS bloated, count(*) AS rows FROM t"
say "Start a writer in the background, then REPACK while it runs."
( while :; do psql -X -q -At -d lesson -c "UPDATE t SET n = n + 1 WHERE id = 10" 2>/dev/null || true; sleep 0.02; done ) & w=$!
sleep 0.3
sql "REPACK (CONCURRENTLY) t"
sleep 0.3; kill $w; wait $w 2>/dev/null || true
sql "SELECT pg_size_pretty(pg_table_size('t')) AS repacked, count(*) AS rows FROM t"
sql "SELECT n AS writes_that_landed FROM t WHERE id = 10"
