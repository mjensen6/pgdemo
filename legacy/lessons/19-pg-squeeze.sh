# 19 pg_squeeze: the extension REPACK grew out of
. "$(dirname "$0")/../lib/helpers.sh"
quiet "CREATE EXTENSION pg_squeeze;
       CREATE TABLE t (id int PRIMARY KEY, pad text) WITH (autovacuum_enabled = false);
       INSERT INTO t SELECT i, repeat('x', 100) FROM generate_series(1, 100000) i;
       DELETE FROM t WHERE id % 10 <> 0;"
quiet "VACUUM t"
sql "SELECT pg_size_pretty(pg_table_size('t')) AS bloated"
say "pg_squeeze works from inside SQL and can also run on a schedule."
sql "SELECT squeeze.squeeze_table('public', 't')"
sql "SELECT pg_size_pretty(pg_table_size('t')) AS squeezed"
