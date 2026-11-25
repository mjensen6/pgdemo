---
title: Materialized views
tldr: Save a slow query's result as a table you can index, and refresh it when you choose, even without blocking readers.
---

```setup
CREATE TABLE orders AS
  SELECT i AS id, i % 500 AS customer, ((i * 37 % 1000) / 10.0)::numeric(8,2) AS amount,
         date '2026-01-01' + i % 90 AS day
  FROM generate_series(1, 300000) i;
ANALYZE orders;
CREATE MATERIALIZED VIEW daily_revenue AS
  SELECT day, count(*) AS orders, sum(amount) AS revenue FROM orders GROUP BY day;
-- The check uses this to compare a materialized view with what it should hold.
CREATE FUNCTION totals_match() RETURNS bool LANGUAGE plpgsql AS $$
BEGIN
  RETURN NOT EXISTS (
    (SELECT customer, sum(amount) FROM orders GROUP BY customer
     EXCEPT SELECT customer, total FROM customer_totals)
    UNION ALL
    (SELECT customer, total FROM customer_totals
     EXCEPT SELECT customer, sum(amount) FROM orders GROUP BY customer));
EXCEPTION WHEN others THEN RETURN false;
END $$;
```

```view
SELECT (SELECT sum(amount) FROM orders) AS orders_revenue,
       (SELECT sum(revenue) FROM daily_revenue) AS view_revenue,
       (SELECT string_agg(matviewname || CASE WHEN ispopulated THEN '' ELSE ' (empty)' END, ', ')
        FROM pg_matviews WHERE schemaname = 'public') AS materialized_views;
```

A normal **view** is a saved query. Every time you read from it, Postgres runs the query again. That's always up to date, but a report that sums 300,000 orders does all that work on every read.

A **materialized view** saves the query's **result**, like a table:

```sql
CREATE MATERIALIZED VIEW daily_revenue AS
  SELECT day, count(*) AS orders, sum(amount) AS revenue FROM orders GROUP BY day;
```

Reading it is as fast as reading a small table, and you can add indexes to it. The cost is that it goes **stale**: new orders don't appear until you run

```sql
REFRESH MATERIALIZED VIEW daily_revenue;
```

which runs the whole query again and replaces the saved result.

**Refreshing without blocking.** A plain refresh locks the view, so anyone reading it waits until the refresh finishes. **`REFRESH MATERIALIZED VIEW CONCURRENTLY`** builds the new result on the side and swaps in only the rows that changed, so readers keep reading. It needs a **unique index** on the view, so it can match old rows to new ones.

**When to use one.** Dashboards and reports that can be a few minutes old, over data that's expensive to summarize. Refresh on a schedule, with pg_cron (lesson 42), or after a batch load. If it must always be current, see the pg_ivm lesson (28), which keeps a view up to date on every change.

**Where it came from.** Materialized views arrived in Postgres 9.3 (2013), and `CONCURRENTLY` in 9.4 (2014). Before that, people built them by hand, with a summary table plus triggers or a scheduled job to rebuild it. Oracle has had them since the late 1990s.

**The Live View** compares the live data with the saved result:

| Column | Meaning |
|---|---|
| `orders_revenue` | Total revenue in `orders`, computed live |
| `view_revenue` | The same total, read from `daily_revenue` |
| `materialized_views` | The materialized views in this playground |

## Try it

Time the report both ways. The query on `orders` reads all 300,000 rows. The materialized view returns the saved result almost at once.

```sql
\timing on
SELECT sum(revenue) FROM (SELECT day, sum(amount) AS revenue FROM orders GROUP BY day) live;
SELECT sum(revenue) FROM daily_revenue;
```

```expect
Timing is on.
Time:
Time:
```

Add an order. In the Live View, `orders_revenue` goes up and `view_revenue` doesn't: the materialized view is stale.

```sql
INSERT INTO orders VALUES (300001, 7, 999.90, '2026-03-31');
```

Refresh it. `view_revenue` catches up.

```sql
REFRESH MATERIALIZED VIEW daily_revenue;
```

```expect
REFRESH MATERIALIZED VIEW
```

To refresh without blocking readers, the view needs a unique index first. `day` is unique in it, one row per day. Then refresh `CONCURRENTLY`.

```sql
CREATE UNIQUE INDEX ON daily_revenue (day);
REFRESH MATERIALIZED VIEW CONCURRENTLY daily_revenue;
```

```expect
CREATE INDEX
REFRESH MATERIALIZED VIEW
```

## Challenge

Create a materialized view called `customer_totals` with one row per customer, columns `customer` and `total` (the sum of `amount`), and make it possible to refresh it without blocking readers.

Hint: CREATE MATERIALIZED VIEW like the intro, grouped by customer, then a unique index on the column that's unique in it.

```solution
CREATE MATERIALIZED VIEW customer_totals AS
  SELECT customer, sum(amount) AS total FROM orders GROUP BY customer;
CREATE UNIQUE INDEX ON customer_totals (customer);
```

```check
SELECT EXISTS (SELECT FROM pg_matviews WHERE matviewname = 'customer_totals' AND ispopulated)
   AND EXISTS (SELECT FROM pg_index WHERE indrelid = to_regclass('customer_totals') AND indisunique)
   AND totals_match();
```
