---
title: Partitioning
tldr: Split one big table into pieces by date or value. Queries read only the pieces they need, and old data drops in an instant.
---

```setup
CREATE TABLE events (at date NOT NULL, msg text) PARTITION BY RANGE (at);
CREATE TABLE events_2024 PARTITION OF events FOR VALUES FROM ('2024-01-01') TO ('2025-01-01');
CREATE TABLE events_2025 PARTITION OF events FOR VALUES FROM ('2025-01-01') TO ('2026-01-01');
INSERT INTO events SELECT date '2024-01-01' + i % 731, 'event ' || i FROM generate_series(1, 100000) i;
ANALYZE events;
```

```view
SELECT c.relname AS partition, pg_get_expr(c.relpartbound, c.oid) AS holds,
       coalesce(n.rows, 0) AS rows, pg_size_pretty(pg_table_size(c.oid)) AS size
FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid
LEFT JOIN (SELECT tableoid, count(*) AS rows FROM events GROUP BY 1) n ON n.tableoid = c.oid
WHERE i.inhparent = 'events'::regclass ORDER BY c.relname;
```

A **partitioned table** looks like one table to your queries, but its rows are stored in separate tables called **partitions**. Each partition holds a slice, chosen by a **partition key**. Here it's a year of the `at` column:

```sql
CREATE TABLE events (at date NOT NULL, msg text) PARTITION BY RANGE (at);
CREATE TABLE events_2025 PARTITION OF events FOR VALUES FROM ('2025-01-01') TO ('2026-01-01');
```

`FROM` includes its value, and `TO` doesn't, so ranges for neighboring years meet without overlapping. An insert goes to the partition that matches. A row that matches no partition is rejected.

**Why split a table?**

- **Partition pruning.** A query that filters on the key reads only the matching partitions and skips the rest.
- **Instant deletes.** Dropping a whole partition is immediate. Deleting the same rows with `DELETE` marks each one dead, which leaves bloat behind and gives VACUUM work, the MVCC lesson's problem.
- **Smaller pieces.** VACUUM, indexes, and maintenance work on one partition at a time.

**The costs.** Queries that don't filter on the key read every partition, and too many partitions slow planning. A unique constraint has to include the key. Partitioning pays off for big tables, think tens of millions of rows, with a natural key like time.

Other ways to split are `PARTITION BY LIST`, for exact values such as a country, and `PARTITION BY HASH`, which spreads rows evenly.

**Where it came from.** Postgres 8.1 (2005) could already partition through table inheritance, but you had to write triggers to send each row to the right table. Postgres 10 (2017) added built-in, **declarative** partitioning, the `PARTITION BY` syntax shown here, and versions 11 to 13 filled in hash partitioning, default partitions, and much faster pruning.

**The Live View** lists the partitions of `events`:

| Column | Meaning |
|---|---|
| `partition` | The partition's own table name |
| `holds` | The range of `at` values it accepts |
| `rows`, `size` | How much data it holds |

## Try it

Try to insert an event from 2026. There's no partition for 2026, so Postgres rejects the row.

```sql
INSERT INTO events VALUES ('2026-03-01', 'new year');
```

Create the 2026 partition, then insert again. A new row appears in the Live View, and the event lands in `events_2026`.

```sql
CREATE TABLE events_2026 PARTITION OF events FOR VALUES FROM ('2026-01-01') TO ('2027-01-01');
INSERT INTO events VALUES ('2026-03-01', 'new year');
```

Query one day. The plan reads only `events_2025`. Postgres pruned the other partitions without looking at them.

```sql
EXPLAIN (COSTS OFF) SELECT count(*) FROM events WHERE at = '2025-06-01';
```

Throw away all of 2024, about 50,000 rows, by dropping its partition. It's instant, and it leaves no dead rows behind.

```sql
DROP TABLE events_2024;
```

## Challenge

Store this event: `('2027-02-14', 'launch')`. Right now there's nowhere for it to go.

Hint: Make room for it the way you made room for 2026, then insert it.

```solution
CREATE TABLE events_2027 PARTITION OF events FOR VALUES FROM ('2027-01-01') TO ('2028-01-01');
INSERT INTO events VALUES ('2027-02-14', 'launch');
```

```check
SELECT EXISTS (SELECT FROM events WHERE at = '2027-02-14' AND msg = 'launch');
```
