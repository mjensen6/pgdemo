---
title: REPACK, new in Postgres 19
tldr: REPACK rebuilds a bloated table to its real size. With CONCURRENTLY, the app keeps writing while it runs.
---

```setup
CREATE TABLE t (id int PRIMARY KEY, n int DEFAULT 0, pad text) WITH (autovacuum_enabled = false);
INSERT INTO t SELECT i, 0, repeat('x', 100) FROM generate_series(1, 200000) i;
DELETE FROM t WHERE id % 10 <> 0;
VACUUM t;
CREATE TABLE events (id int PRIMARY KEY, pad text) WITH (autovacuum_enabled = false);
INSERT INTO events SELECT i, repeat('x', 100) FROM generate_series(1, 100000) i;
DELETE FROM events WHERE id % 10 <> 0;
VACUUM events;
```

```view
SELECT pg_size_pretty(pg_table_size('t')) AS t_size,
       (SELECT count(*) FROM t) AS t_rows,
       (SELECT n FROM t WHERE id = 10) AS writes_landed,
       pg_size_pretty(pg_table_size('events')) AS events_size;
```

Table `t` once held 200,000 rows. 90% were deleted and vacuumed, but the file kept its full size.

```sql
SELECT pg_size_pretty(pg_table_size('t')) AS size, count(*) AS rows FROM t;
```

Now an app writes to the table nonstop while `REPACK (CONCURRENTLY)` rebuilds it. Unlike `VACUUM FULL`, it doesn't lock the app out, so the writes keep landing during the rebuild.

```sh
# A pretend app: update row 10 every 20 ms.
(while :; do psql -qc "UPDATE t SET n = n + 1 WHERE id = 10"; sleep 0.02; done) &>/dev/null &
sleep 1; psql -Atc "SELECT 'writes so far: ' || n FROM t WHERE id = 10"
psql -c "REPACK (CONCURRENTLY) t"
sleep 1; kill $!; wait $! 2>/dev/null
psql -Atc "SELECT 'writes so far: ' || n FROM t WHERE id = 10"
```

The table is back to its real size, and no rows or writes were lost.

```sql
SELECT pg_size_pretty(pg_table_size('t')) AS size, count(*) AS rows FROM t;
```

## Challenge

Table `events` is bloated too. Shrink it to its real size without blocking writers.

Hint: The same command works on any table with a primary key. Add the option in parentheses.

```solution
REPACK (CONCURRENTLY) events;
```

```check
SELECT pg_table_size('events') < 3000000;
```
