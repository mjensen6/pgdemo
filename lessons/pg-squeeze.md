---
title: pg_squeeze
tldr: Rebuild a bloated table online from inside SQL, using logical decoding. It's the extension REPACK CONCURRENTLY grew out of.
---

```setup
CREATE EXTENSION pg_squeeze;
CREATE TABLE t (id int PRIMARY KEY, pad text) WITH (autovacuum_enabled = false);
INSERT INTO t SELECT i, repeat('x', 100) FROM generate_series(1, 100000) i;
DELETE FROM t WHERE id % 10 <> 0;
VACUUM t;
CREATE TABLE events (id int PRIMARY KEY, pad text) WITH (autovacuum_enabled = false);
INSERT INTO events SELECT i, repeat('x', 100) FROM generate_series(1, 100000) i;
DELETE FROM events WHERE id % 10 <> 0;
VACUUM events;
```

```view
SELECT relname AS "table", n_live_tup AS rows, pg_size_pretty(pg_table_size(relid)) AS size
FROM pg_stat_user_tables WHERE relname IN ('t', 'events') ORDER BY 1;
```

**pg_squeeze** does the same job as pg_repack, from the last lesson: it rebuilds a bloated table to its real size while the app keeps working. The differences are where you run it and how it keeps up with changes.

**It runs from SQL.** There's no separate program. You call a function:

```sql
SELECT squeeze.squeeze_table('public', 't');
```

The arguments are the schema and the table. It can also run on a schedule by itself: add a row to `squeeze.tables` saying which table and when, and a background worker squeezes it at those times.

**It uses logical decoding instead of triggers.** Every change in Postgres is first written to the **write-ahead log** (WAL), which exists so the server can recover after a crash. **Logical decoding** reads that log back as a stream of row changes. pg_squeeze copies the table, reads the changes made meanwhile from the WAL, applies them to the copy, and swaps the two with a brief lock. Since it adds no triggers, ordinary writes don't get any slower while it runs.

**The requirements.** The table needs a primary key, and the disk needs room for a copy. The server needs `wal_level` set to at least `replica`, and pg_squeeze must be loaded at startup through `shared_preload_libraries`. This course's server is already set up for it.

**Where it came from.** pg_squeeze is a third-party extension from Cybertec, written by Antonin Houska. In Postgres 19 (2026), his design moved into Postgres itself as `REPACK (CONCURRENTLY)`, which the REPACK lesson showed. On older versions, pg_squeeze and pg_repack remain the tools for the job.

**The Live View** shows two bloated tables, 90% deleted and vacuumed, still at full size.

## Try it

Table `t` holds 10,000 rows but takes the space of 100,000.

```sql
SELECT pg_size_pretty(pg_table_size('t')) AS size, count(*) AS rows FROM t;
```

Squeeze it. Watch its `size` drop in the Live View.

```sql
SELECT squeeze.squeeze_table('public', 't');
```

It's back to its real size, with every row still there.

```sql
SELECT pg_size_pretty(pg_table_size('t')) AS size, count(*) AS rows FROM t;
```

## Challenge

Table `events` is bloated too. Shrink it with pg_squeeze.

Hint: Call the same function with the other table's name.

```solution
SELECT squeeze.squeeze_table('public', 'events');
```

```check
SELECT pg_table_size('events') < 3000000;
```
