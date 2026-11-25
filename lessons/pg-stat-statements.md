---
title: pg_stat_statements
tldr: Postgres keeps a running tally of every query shape, so you can see which ones use the most time and fix those first.
---

```setup
CREATE EXTENSION pg_stat_statements;
CREATE TABLE orders AS
  SELECT i AS id, i % 1000 AS customer, 'user' || i || '@example.com' AS email
  FROM generate_series(1, 200000) i;
ALTER TABLE orders ADD PRIMARY KEY (id);
ANALYZE orders;
SELECT pg_stat_statements_reset(0, (SELECT oid FROM pg_database WHERE datname = current_database()), 0) IS NOT NULL AS reset;
-- A pretend app's traffic from earlier today.
SELECT * FROM orders WHERE id = 17;
SELECT * FROM orders WHERE id = 4242;
SELECT * FROM orders WHERE id = 99;
SELECT * FROM orders WHERE email = 'user17@example.com';
SELECT * FROM orders WHERE email = 'user4242@example.com';
SELECT * FROM orders WHERE email = 'user99@example.com';
SELECT * FROM orders WHERE email = 'user123@example.com';
SELECT count(*) FROM orders WHERE customer = 7;
```

```view
SELECT calls, round(total_exec_time::numeric, 1) AS total_ms, round(mean_exec_time::numeric, 2) AS mean_ms,
       left(regexp_replace(query, '\s+', ' ', 'g'), 60) AS query
FROM pg_stat_statements
WHERE dbid = (SELECT oid FROM pg_database WHERE datname = current_database())
  AND query NOT LIKE '%pg_stat%' AND query ~* '^\s*select'
ORDER BY total_exec_time DESC LIMIT 6;
```

When an app is slow, the hard part is knowing **which** query to fix. A query that takes 2 seconds once a day matters less than one that takes 20 milliseconds a million times.

**pg_stat_statements** is an extension that records every statement the server runs. It groups statements that differ only in their values: `WHERE id = 17` and `WHERE id = 4242` become one entry, `WHERE id = $1`. For each, it keeps running totals:

| Column | Meaning |
|---|---|
| `calls` | How many times it ran |
| `total_exec_time` | Time spent in it altogether, in milliseconds |
| `mean_exec_time` | Average time per call |
| `rows` | Rows returned or changed, altogether |

Sort by `total_exec_time` to find where the server's time actually goes. Then use `EXPLAIN` on the worst one to see why.

**The cost** is small: a little memory and a little work per statement. Most hosted Postgres services turn it on by default. It has to be loaded when the server starts, through `shared_preload_libraries`, so turning it on takes a restart. This course's server already has it.

**Where it came from.** It's a contrib extension that ships with Postgres, added in version 8.4 (2009). Version 9.2 (2012) made it group queries by shape, which made it really useful. It was never moved into core, but it's on almost every production server.

**The Live View** shows the queries in this playground that used the most time, worst first, using the columns above.

## Try it

Here's the pretend app's traffic, with the most time used first. The lookup by `email` is the worst offender, even though the lookup by `id` ran almost as often.

```sql
SELECT calls, round(total_exec_time::numeric, 2) AS total_ms, round(mean_exec_time::numeric, 2) AS mean_ms, query
FROM pg_stat_statements
WHERE dbid = (SELECT oid FROM pg_database WHERE datname = current_database()) AND query LIKE 'SELECT%FROM orders%'
ORDER BY total_exec_time DESC;
```

```expect
SELECT * FROM orders WHERE email = $1
SELECT * FROM orders WHERE id = $1
```

Why is it slow? `EXPLAIN` shows it reads the whole table (`Seq Scan`), because there's no index on `email`. The lookup by `id` uses the primary key's index instead.

```sql
EXPLAIN (COSTS OFF) SELECT * FROM orders WHERE email = 'user17@example.com';
EXPLAIN (COSTS OFF) SELECT * FROM orders WHERE id = 17;
```

```expect
Seq Scan on orders
Index Scan using orders_pkey
```

Run a few more lookups by email with different values. In the Live View, they're added to the same entry: `calls` goes up and `total_ms` grows.

```sql
SELECT id FROM orders WHERE email = 'user1@example.com';
SELECT id FROM orders WHERE email = 'user2@example.com';
```

## Challenge

Fix the query that uses the most total time, so it no longer reads the whole table.

Hint: The EXPLAIN step showed why it's slow. An index on the column it searches fixes it.

```solution
CREATE INDEX ON orders (email);
```

```check
-- Some index on orders starts with email.
SELECT EXISTS (SELECT FROM pg_index
               WHERE indrelid = 'orders'::regclass
                 AND indkey[0] = (SELECT attnum FROM pg_attribute WHERE attrelid = 'orders'::regclass AND attname = 'email'));
```
