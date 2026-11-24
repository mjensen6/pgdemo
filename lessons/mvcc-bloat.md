---
title: MVCC and bloat
tldr: Updates leave dead rows behind. VACUUM reuses the space but rarely shrinks the file.
---

```setup
CREATE EXTENSION pgstattuple;
CREATE TABLE t (id int, note text) WITH (autovacuum_enabled = false);
INSERT INTO t SELECT i, 'hello' FROM generate_series(1, 10000) i;
```

```view
SELECT pg_size_pretty(pg_table_size('t')) AS size, tuple_count AS rows,
       dead_tuple_count AS dead, round(free_percent) AS free_pct
FROM pgstattuple('t');
```

Postgres never changes a row in place. An `UPDATE` writes a brand-new copy of the row and marks the old copy as expired. A `DELETE` only marks the row as expired. This is **MVCC**, short for multi-version concurrency control.

**Why work this way?** A query that started before your change keeps reading the old copy, so it sees one consistent snapshot. Readers never wait for writers, and writers never wait for readers.

**The cost** is that expired copies, called **dead rows**, stay in the table's file and take up space. **VACUUM** clears them out. Postgres normally runs it for you in the background, a feature called autovacuum, but it's switched off for this lesson's table so you can watch.

VACUUM marks the space as free for new rows to reuse, but it almost never hands the space back to the operating system. A table that once grew stays big. That leftover empty space is called **bloat**.

**The Live View** watches table `t`, which starts with 10,000 rows:

| Column | Meaning |
|---|---|
| `size` | Disk space the table takes |
| `rows` | Live rows, the ones queries can see |
| `dead` | Dead rows waiting for VACUUM |
| `free_pct` | How much of the file is empty, reusable space |

## Try it

Update every row in the table. Postgres writes 10,000 new copies and keeps the 10,000 old ones as dead rows, so in the Live View `dead` jumps to 10,000 and `size` roughly doubles.

```sql
UPDATE t SET note = 'world';
```

VACUUM clears the dead rows. In the Live View, `dead` drops to 0 and `free_pct` jumps to about 50, but `size` stays the same. The space is free inside the file, not given back. That's bloat.

```sql
VACUUM t;
```

Update every row again. This time the new copies fill the free space VACUUM left behind, so `size` barely moves. VACUUM's space gets reused. It just doesn't shrink the file.

```sql
UPDATE t SET note = 'again';
```

## Challenge

Shrink table `t` back down to its original size.

Hint: There's a command new in Postgres 19 that rebuilds a table.

```solution
REPACK t;
```

```check
-- Shrunk after the updates, not just untouched since setup.
SELECT pg_table_size('t') < 600000 AND NOT EXISTS (SELECT FROM t WHERE note = 'hello');
```
