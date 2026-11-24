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

Table `t` has 10,000 rows. Watch the Live View. An UPDATE writes a new version of each row and leaves the old one behind as a dead row, so the table doubles in size.

```sql
UPDATE t SET note = 'world';
```

VACUUM clears the dead rows so their space can be reused. The dead count drops to zero, but the file stays the same size. The empty space inside it is called bloat.

```sql
VACUUM t;
```

## Challenge

Shrink table `t` back down to its original size.

Hint: There's a command new in Postgres 19 that rebuilds a table.

```solution
REPACK t;
```

```check
-- Shrunk after the update, not just untouched since setup.
SELECT pg_table_size('t') < 600000 AND NOT EXISTS (SELECT FROM t WHERE note = 'hello');
```
