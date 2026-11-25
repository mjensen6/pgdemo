---
title: pg_partman
tldr: An extension that creates future partitions before you need them and drops old ones on schedule, so a partitioned table runs itself.
---

```setup
CREATE SCHEMA partman;
CREATE EXTENSION pg_partman SCHEMA partman;
CREATE TABLE metrics (at timestamptz NOT NULL, value int) PARTITION BY RANGE (at);
SELECT partman.create_parent(p_parent_table := 'public.metrics', p_control := 'at', p_interval := '1 month',
                             p_premake := 2, p_start_partition := (date_trunc('month', now()) - interval '6 months')::text);
UPDATE partman.part_config SET infinite_time_partitions = true WHERE parent_table = 'public.metrics';
INSERT INTO metrics SELECT now() - i * interval '1 hour', i FROM generate_series(0, 24 * 180) i;
```

```view
SELECT c.relname AS partition, coalesce(n.rows, 0) AS rows
FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid
LEFT JOIN (SELECT tableoid, count(*) AS rows FROM metrics GROUP BY 1) n ON n.tableoid = c.oid
WHERE i.inhparent = 'metrics'::regclass ORDER BY c.relname;
```

The partitioning lesson showed the catch with time-based partitions: someone has to create next month's partition before next month starts, or inserts fail. And someone has to drop old partitions when they're no longer needed. Forget either one, and you find out the hard way.

**pg_partman** is an extension that does both. You register a partitioned table once, and from then on it keeps the partitions in shape:

```sql
SELECT partman.create_parent(p_parent_table := 'public.metrics', p_control := 'at', p_interval := '1 month');
```

Its settings for each table are rows in the table **`partman.part_config`**. You change them with a plain `UPDATE`:

| Column | Meaning |
|---|---|
| `premake` | How many partitions to keep ready ahead of now |
| `retention` | How old a partition may get before it's dropped, such as `'3 months'`. Empty means keep everything |
| `retention_keep_table` | `true` only detaches old partitions, keeping them as plain tables. `false` drops them |
| `infinite_time_partitions` | Keep making future partitions even when no new data is arriving. It's on here |

**`CALL partman.run_maintenance_proc()`** applies the settings: it creates what's missing and drops what's too old. In production you run it on a schedule, with pg_cron (lesson 42) or pg_partman's own background worker, so nobody has to remember.

It also creates a **default partition**, `metrics_default`, which catches any row that matches no other partition instead of rejecting it.

**Where it came from.** pg_partman is a third-party extension by Keith Fiske. It started when partitioning in Postgres meant triggers and inheritance, and it did the trigger-writing for you. Built-in partitioning (Postgres 10, 2017) took over the routing of rows, but not creating and dropping partitions over time, so pg_partman stayed essential. Since version 5 (2023) it manages only built-in partitioning.

**The Live View** lists the partitions of `metrics`, oldest first, with how many rows each holds. The data covers the last 180 days.

## Try it

Look at the settings for `metrics`. It keeps 2 partitions ready ahead of now, and has no retention, so nothing is ever dropped.

```sql
SELECT parent_table, partition_interval, premake, retention, retention_keep_table FROM partman.part_config;
```

```expect
public.metrics
1 mon
2
```

Keep 4 months ready instead of 2, then run maintenance. More future partitions appear at the bottom of the Live View, with 0 rows waiting for their month.

```sql
UPDATE partman.part_config SET premake = 4 WHERE parent_table = 'public.metrics';
CALL partman.run_maintenance_proc();
```

```expect
UPDATE 1
CALL
```

Keep only 3 months of history and drop anything older. After maintenance, the oldest partitions disappear from the top of the Live View, data and all.

```sql
UPDATE partman.part_config SET retention = '3 months', retention_keep_table = false WHERE parent_table = 'public.metrics';
CALL partman.run_maintenance_proc();
```

```expect
UPDATE 1
CALL
```

## Challenge

Change `metrics` to keep 6 months of partitions ready ahead, and only 1 month of history. Then apply it.

Hint: Same columns as the last two steps, then the same CALL.

```solution
UPDATE partman.part_config SET premake = 6, retention = '1 month' WHERE parent_table = 'public.metrics';
CALL partman.run_maintenance_proc();
```

```check
-- Partition names end in their start date, like metrics_p20261101.
WITH p AS (
  SELECT to_date(substring(c.relname FROM 'p(\d{8})$'), 'YYYYMMDD') AS starts
  FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid
  WHERE i.inhparent = 'metrics'::regclass AND c.relname ~ 'p\d{8}$')
SELECT max(starts) >= date_trunc('month', now()) + interval '6 months'
   AND min(starts) >= date_trunc('month', now()) - interval '1 month'
FROM p;
```
