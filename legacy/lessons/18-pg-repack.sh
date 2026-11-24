# 18 pg_repack: the classic extension for older versions
. "$(dirname "$0")/../lib/helpers.sh"
quiet "CREATE EXTENSION pg_repack;
       CREATE TABLE t (id int PRIMARY KEY, pad text) WITH (autovacuum_enabled = false);
       INSERT INTO t SELECT i, repeat('x', 100) FROM generate_series(1, 100000) i;
       DELETE FROM t WHERE id % 10 <> 0;"
quiet "VACUUM t"
sql "SELECT pg_size_pretty(pg_table_size('t')) AS bloated"
say "pg_repack is a command-line tool. It uses triggers to capture writes during the rewrite."
echo 'pg_repack -d lesson -t t'
pg_repack -d lesson -t t 2>&1
sql "SELECT pg_size_pretty(pg_table_size('t')) AS repacked"
