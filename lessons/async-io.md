---
title: Asynchronous I/O, new in Postgres 18
tldr: Postgres now asks for many pages at once instead of waiting for each one, which speeds up big scans on slow disks.
---

```setup
CREATE EXTENSION pg_buffercache;
CREATE TABLE big AS SELECT i, repeat('x', 100) AS pad FROM generate_series(1, 500000) i;
VACUUM ANALYZE big;
```

```view
SELECT current_setting('io_method') AS io_method,
       current_setting('io_min_workers') || ' to ' || current_setting('io_max_workers') AS io_workers,
       pg_size_pretty(pg_table_size('big')) AS table_size,
       coalesce((SELECT array_to_string(setconfig, ', ') FROM pg_db_role_setting
                 WHERE setrole = 0 AND setdatabase = (SELECT oid FROM pg_database WHERE datname = current_database())),
                '(none)') AS playground_defaults;
```

When a query needs a table page that isn't in memory, Postgres reads it from disk. Before Postgres 18, it mostly asked for one page and waited, then asked for the next and waited again. On a fast local SSD each wait is tiny. On cloud storage, where every read crosses a network, the waits add up to most of the query's time.

**Asynchronous I/O** changes that. Postgres asks for many pages ahead of time and keeps working while they arrive, so the disk is always busy with the next pages.

Three settings control it:

| Setting | Meaning |
|---|---|
| `io_method` | How reads are issued. `worker` (the default) hands them to background I/O workers. `io_uring` uses a fast Linux interface. `sync` is the old behavior |
| `effective_io_concurrency` | How many reads may be in flight at once. `0` turns read-ahead off |
| `io_combine_limit` | How many neighboring pages get combined into one bigger read |

**Settings live at different levels.** `SET` changes a setting for your session only, until you disconnect. `ALTER DATABASE name SET ...` makes it the default for every new session in that database. Server-wide ones like `io_method` are set when the server starts.

**What to expect.** The gain depends on how slow your storage is. Here, the operating system still caches the file even after Postgres drops it from its own cache, so the difference is small. On network disks, scans are often several times faster. Writes still happen the old way for now.

**Where it came from.** Asynchronous I/O arrived in Postgres 18 (2025), after years of work led by Andres Freund. Before that, the only read-ahead was a hint to the operating system for some index scans, which is what `effective_io_concurrency` controlled since Postgres 8.4 (2009). It now controls the new system.

**The Live View** shows:

| Column | Meaning |
|---|---|
| `io_method`, `io_workers` | How this server issues reads, and how many I/O workers it may run. Postgres 19 starts more workers when reads pile up, between `io_min_workers` and `io_max_workers` |
| `table_size` | Size of table `big` |
| `playground_defaults` | Settings saved for new sessions in this playground with `ALTER DATABASE` |

## Try it

Turn on timing, and keep the comparison fair by using one process per scan. `pg_buffercache_evict_relation` drops table `big` from Postgres's memory, so the next scan has to read it again. Then scan it with read-ahead off.

```sql
\timing on
SET max_parallel_workers_per_gather = 0;
SET effective_io_concurrency = 0;
SELECT buffers_evicted > 0 AS evicted FROM pg_buffercache_evict_relation('big');
SELECT count(*) FROM big;
```

```expect
500000
Time:
```

Same again with read-ahead back on, at the default of 16 reads in flight. Compare the `Time:` of the two `count(*)` lines. It should be faster, by more on slower disks.

```sql
RESET effective_io_concurrency;
SELECT buffers_evicted > 0 AS evicted FROM pg_buffercache_evict_relation('big');
SELECT count(*) FROM big;
```

```expect
500000
Time:
```

`SET` only lasted for this session. To make a setting the default for every new session in this playground, save it on the database. Watch `playground_defaults` in the Live View.

```sql
ALTER DATABASE lesson_async_io SET effective_io_concurrency = 0;
```

```expect
ALTER DATABASE
```

## Challenge

Saving `effective_io_concurrency = 0` turned read-ahead off for every new session. Change the playground's saved default so new sessions keep 64 reads in flight instead.

Hint: Run the last step's command again with a different number.

```solution
ALTER DATABASE lesson_async_io SET effective_io_concurrency = 64;
```

```check
SELECT coalesce('effective_io_concurrency=64' = ANY (setconfig), false)
FROM pg_database d LEFT JOIN pg_db_role_setting s ON s.setdatabase = d.oid AND s.setrole = 0
WHERE d.datname = current_database();
```
