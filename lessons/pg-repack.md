---
title: pg_repack
tldr: The classic extension for rebuilding a bloated table while the app keeps running, on any Postgres version, not just 19.
---

```setup
CREATE EXTENSION pg_repack;
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

The REPACK lesson rebuilt a bloated table without blocking the app, with a command new in Postgres 19. Most production databases run older versions, and for those the tool has been **pg_repack** for over a decade.

It has two parts:

- **An extension** in the database, created with `CREATE EXTENSION pg_repack`. It holds the functions that do the work.
- **A command-line program**, `pg_repack`, that you run from a shell. It drives the rebuild step by step from outside.

```
pg_repack -d mydb -t mytable
```

**How it works.** It creates an empty copy of the table and adds a **trigger** to the original, which records every insert, update, and delete in a log table. It copies the live rows into the new table, packed tightly, builds the indexes, replays the logged changes, and then swaps the two tables with a brief lock. The app keeps reading and writing the whole time.

**The requirements** are the same as `REPACK (CONCURRENTLY)`. The table needs a primary key, so changes can be matched to rows, and the disk needs room for a second copy while it runs. The triggers add a little overhead to every write while it runs.

**Where it came from.** It started as `pg_reorg`, from NTT in Japan. When that stalled, `pg_repack` forked from it in 2012 and became the standard tool for removing bloat online. `pg_squeeze` (next lesson) later did the same job with logical decoding instead of triggers, and that approach became `REPACK (CONCURRENTLY)` in Postgres 19.

**Running a shell command from SQL.** The SQL pane can run a shell command with `\!`, as in `\! pg_repack -d lesson_pg_repack -t t`. The steps below run in a shell, so they don't need it.

**The Live View** shows two bloated tables. 90% of their rows were deleted and vacuumed, but the files kept their full size:

| Column | Meaning |
|---|---|
| `rows` | Live rows |
| `size` | Disk space the table takes |

## Try it

Table `t` holds 10,000 rows but takes the space of 100,000.

```sql
SELECT pg_size_pretty(pg_table_size('t')) AS size, count(*) AS rows FROM t;
```

Rebuild it with the `pg_repack` program. `-d` names the database and `-t` the table. Watch `size` for `t` drop in the Live View.

```sh
pg_repack -d lesson_pg_repack -t t
```

It's back to its real size, with every row still there.

```sql
SELECT pg_size_pretty(pg_table_size('t')) AS size, count(*) AS rows FROM t;
```

## Challenge

Table `events` is bloated too. Shrink it with pg_repack, from the SQL pane.

Hint: Put \! in front of the same command you ran in the shell, with the other table's name.

```solution
\! pg_repack -d lesson_pg_repack -t events
```

```check
SELECT pg_table_size('events') < 3000000;
```
