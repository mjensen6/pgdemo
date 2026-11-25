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

In the MVCC lesson you saw that VACUUM frees the space dead rows leave behind but almost never shrinks the file. A table that once grew stays big. That empty space is **bloat**. It wastes disk, and it slows down queries that scan the whole table, because they read the empty pages too.

**REPACK** fixes bloat by rebuilding the table. It copies the live rows into a brand-new file, packed tightly, rebuilds the indexes, then swaps the new file in and deletes the old one. The table ends up at its real size.

**The catch with a plain rebuild** is the lock. `REPACK t` locks the table for the whole rebuild, so the app can't read or write it until the rebuild finishes. On a big table that can take minutes or hours.

**`REPACK (CONCURRENTLY)`** avoids that. It copies the rows while the app keeps working, records every change made during the copy, and replays those changes onto the new file. It only locks the table for a moment at the very end to swap the files. It has a few requirements:

- The table needs a primary key, so changes can be matched to the right rows.
- It can't run inside a transaction block.
- It needs free disk space for the new copy while the old one still exists.

**Where it came from.** `REPACK` is new in Postgres 19 (2026). Before that, the built-in ways to rebuild a table were `VACUUM FULL` and `CLUSTER`, and both lock the table for the whole rebuild. They still work, but `REPACK` now does both jobs. To rebuild without blocking the app, people installed extensions for more than a decade. The first was `pg_reorg`, then its fork `pg_repack` (2012), which became the standard tool. Later came `pg_squeeze`. The built-in `REPACK (CONCURRENTLY)` is largely `pg_squeeze` moved into Postgres. It was designed and written by the same author, and it tracks changes the same way, through the write-ahead log rather than triggers.

**The Live View** watches two bloated tables:

| Column | Meaning |
|---|---|
| `t_size` | Disk space table `t` takes |
| `t_rows` | Live rows in `t` |
| `writes_landed` | How many times the pretend app has updated row 10 |
| `events_size` | Disk space table `events` takes, for the challenge |

## Try it

Table `t` once held 200,000 rows. 90% were deleted and vacuumed, but the file kept its full size. Compare the size with the row count: about 20,000 small rows shouldn't need this much space.

```sql
SELECT pg_size_pretty(pg_table_size('t')) AS size, count(*) AS rows FROM t;
```

```expect
27 MB | 20000
```

Now a pretend app updates row 10 every 20 ms while `REPACK (CONCURRENTLY)` rebuilds the table. In the Live View, watch `writes_landed` keep climbing during the rebuild. The app is never locked out. The step prints the write count before and after, so you can see writes landed during the rebuild.

```sh
# A pretend app: update row 10 every 20 ms.
(while :; do psql -qc "UPDATE t SET n = n + 1 WHERE id = 10"; sleep 0.02; done) &>/dev/null &
sleep 1; psql -Atc "SELECT 'writes so far: ' || n FROM t WHERE id = 10"
psql -c "REPACK (CONCURRENTLY) t"
sleep 1; kill $!; wait $! 2>/dev/null
psql -Atc "SELECT 'writes so far: ' || n FROM t WHERE id = 10"
```

```expect
writes so far:
REPACK
writes so far:
```

The table is back to its real size, about a tenth of what it was, and no rows or writes were lost. `t_size` in the Live View dropped the same way.

```sql
SELECT pg_size_pretty(pg_table_size('t')) AS size, count(*) AS rows FROM t;
```

```expect
| 20000
```

## Challenge

Table `events` is bloated too. Shrink it to its real size without blocking writers.

Hint: Use the same command you just ran on `t`, with the option that keeps the app running.

```solution
REPACK (CONCURRENTLY) events;
```

```check
SELECT pg_table_size('events') < 3000000;
```
