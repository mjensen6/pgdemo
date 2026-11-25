---
title: HypoPG
tldr: Ask the planner whether an index would help before you spend the time and disk to build it.
---

```setup
CREATE EXTENSION hypopg;
CREATE TABLE signups AS
  SELECT i AS id, 'user' || i || '@example.com' AS email, i % 40 AS country,
         date '2020-01-01' + i % 2000 AS day
  FROM generate_series(1, 400000) i;
ANALYZE signups;
```

```view
SELECT 'real' AS kind, indexrelid::regclass::text AS index, pg_size_pretty(pg_relation_size(indexrelid)) AS size
FROM pg_index WHERE indrelid = 'signups'::regclass
UNION ALL
SELECT 'none yet', '', '' WHERE NOT EXISTS (SELECT FROM pg_index WHERE indrelid = 'signups'::regclass);
```

Building an index on a big table takes time, disk space, and slows every write from then on. Before you build one, you want to know: would the planner even **use** it for the query you're trying to fix?

**HypoPG** is an extension that creates **hypothetical indexes**. They don't exist on disk and take no time to build. They exist only in your session's memory, but the planner believes in them. So plain `EXPLAIN` shows the plan you'd get if the index were real:

```sql
SELECT * FROM hypopg_create_index('CREATE INDEX ON signups (email)');
EXPLAIN SELECT * FROM signups WHERE email = 'user42@example.com';
```

If the plan uses it, the index is worth building. If the planner ignores it, you've saved yourself the trouble.

| Function | Does |
|---|---|
| `hypopg_create_index('CREATE INDEX ...')` | Add a hypothetical index |
| `hypopg_list_indexes` | A view listing them |
| `hypopg_relation_size(indexrelid)` | Estimate how big the real index would be |
| `hypopg_reset()` | Remove them all |

**The limits.** It's only the planner's opinion. `EXPLAIN ANALYZE` actually runs the query, so it ignores hypothetical indexes. Other sessions never see them, and they vanish when you disconnect. Once you've chosen, build the real index and measure.

**Where it came from.** HypoPG is a third-party extension written by Julien Rouhaud and sponsored by Dalibo, first released in 2015. It's not part of core Postgres, but tools that suggest indexes, and some hosted Postgres services, use it under the hood.

**The Live View** lists the **real** indexes on `signups`, which has 400,000 rows. Hypothetical ones never appear there, because they aren't real.

## Try it

A lookup by email reads the whole table today. The plan says `Seq Scan`.

```sql
EXPLAIN SELECT * FROM signups WHERE email = 'user42@example.com';
```

```expect
Seq Scan on signups
```

Create a hypothetical index on `email` and ask again. The plan now uses it, `Index Scan using <...>btree_signups_email`, and its cost drops from thousands to single digits. The Live View still shows no real index.

```sql
SELECT * FROM hypopg_create_index('CREATE INDEX ON signups (email)');
EXPLAIN SELECT * FROM signups WHERE email = 'user42@example.com';
```

```expect
btree_signups_email
Index Scan using
btree_signups_email
```

How big would it be if you built it?

```sql
SELECT index_name, pg_size_pretty(hypopg_relation_size(indexrelid)) AS estimated_size FROM hypopg_list_indexes;
```

```expect
btree_signups_email
```

`EXPLAIN ANALYZE` runs the query for real, so it can't use an index that doesn't exist. Back to `Seq Scan`. Then clear the hypothetical indexes.

```sql
EXPLAIN ANALYZE SELECT * FROM signups WHERE email = 'user42@example.com';
SELECT hypopg_reset();
```

```expect
Seq Scan on signups
```

## Challenge

This report is slow: `SELECT count(*) FROM signups WHERE country = 7 AND day = '2024-06-01'`. Use HypoPG to find out which **single-column** index the planner would choose for it, `country` or `day`, then build that one for real.

Hint: Create both hypothetical indexes, EXPLAIN the report, and see which one the plan names. Then CREATE INDEX on that column.

```solution
SELECT * FROM hypopg_create_index('CREATE INDEX ON signups (country)');
SELECT * FROM hypopg_create_index('CREATE INDEX ON signups (day)');
EXPLAIN SELECT count(*) FROM signups WHERE country = 7 AND day = '2024-06-01';
SELECT hypopg_reset();
CREATE INDEX ON signups (day);
```

```check
-- day has 2,000 values and country only 40, so day narrows it down far more.
SELECT EXISTS (SELECT FROM pg_index
               WHERE indrelid = 'signups'::regclass AND indnatts = 1
                 AND indkey[0] = (SELECT attnum FROM pg_attribute WHERE attrelid = 'signups'::regclass AND attname = 'day'));
```
