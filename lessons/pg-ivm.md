---
title: pg_ivm
tldr: A materialized view that keeps itself up to date on every change, so it's fast to read and never stale.
---

```setup
CREATE EXTENSION pg_ivm;
CREATE TABLE sales AS
  SELECT i AS id, i % 10 AS shop, 'product ' || i % 25 AS product, 5 + i % 20 AS amount
  FROM generate_series(1, 200000) i;
SELECT pgivm.create_immv('shop_totals', 'SELECT shop, sum(amount) AS total, count(*) AS sales FROM sales GROUP BY shop');
CREATE MATERIALIZED VIEW shop_totals_mv AS SELECT shop, sum(amount) AS total, count(*) AS sales FROM sales GROUP BY shop;
-- The check uses this to compare an IMMV with what it should hold.
CREATE FUNCTION product_counts_match() RETURNS bool LANGUAGE plpgsql AS $$
BEGIN
  RETURN NOT EXISTS (
    (SELECT product, count(*) FROM sales GROUP BY product EXCEPT SELECT product, sales FROM product_counts)
    UNION ALL
    (SELECT product, sales FROM product_counts EXCEPT SELECT product, count(*) FROM sales GROUP BY product));
EXCEPTION WHEN others THEN RETURN false;
END $$;
```

```view
SELECT 'live query' AS source, (SELECT sum(amount) FROM sales WHERE shop = 3) AS shop_3_total
UNION ALL SELECT 'IMMV shop_totals', (SELECT total FROM shop_totals WHERE shop = 3)
UNION ALL SELECT 'matview shop_totals_mv', (SELECT total FROM shop_totals_mv WHERE shop = 3)
UNION ALL SELECT 'IMMVs here: ' || string_agg(immvrelid::text, ', '), NULL FROM pgivm.pg_ivm_immv;
```

A materialized view (lesson 9) saves a slow query's result, so reading it is fast. But it goes stale the moment the data changes, until someone runs `REFRESH`, which redoes the whole query.

**Incremental view maintenance** (IVM) keeps the saved result current as you go. When a row is inserted, updated, or deleted, only the part of the result it affects is updated. Add a sale to shop 3, and only shop 3's total changes.

**pg_ivm** is an extension that does this. You create an **IMMV**, an incrementally maintained materialized view, with a function:

```sql
SELECT pgivm.create_immv('shop_totals', 'SELECT shop, sum(amount) AS total, count(*) AS sales FROM sales GROUP BY shop');
```

It adds triggers to the tables in the query, and from then on every change updates the IMMV within the same transaction. Reading it is as fast as reading a small table, and it's never stale. It also adds a few hidden helper columns, with names starting `__ivm_`, that it needs to do the updates.

**The costs.** Every write to the base tables now also updates the IMMV, so writes get slower, and busy writers to the same group can wait on each other. Only some queries are supported: joins, `GROUP BY`, `DISTINCT`, and aggregates like `count`, `sum`, `avg`, `min`, and `max` work. Window functions, `ORDER BY`, and many subqueries don't. For dashboards that can be a few minutes old, a plain materialized view refreshed on a schedule is often the simpler choice.

**Where it came from.** pg_ivm is a third-party extension from SRA OSS, first released in 2022. It came out of a patch that Yugo Nagata of SRA OSS has proposed for core Postgres since 2018. That isn't in core yet, so the extension makes the feature available now.

**The Live View** compares shop 3's total three ways: computed live from `sales`, read from the IMMV, and read from an ordinary materialized view. It also lists the IMMVs in this playground.

## Try it

Add a big sale to shop 3. In the Live View, the IMMV changes with the live total at once. The ordinary materialized view doesn't change, since it's stale until refreshed.

```sql
INSERT INTO sales VALUES (200001, 3, 'product 1', 1000);
```

```expect
INSERT 0 1
```

Deletes and updates count too. Remove that sale, and the IMMV goes straight back.

```sql
DELETE FROM sales WHERE id = 200001;
```

```expect
DELETE 1
```

Look inside the IMMV. Next to your columns are hidden helper columns, which pg_ivm uses to adjust sums and counts without rereading `sales`.

```sql
SELECT * FROM shop_totals WHERE shop = 3;
```

```expect
__ivm_count
260000
```

## Challenge

Create an IMMV called `product_counts` with one row per product: columns `product` and `sales`, the number of sales of it.

Hint: The intro's create_immv call, with a new name and a query grouped by product, using count(*) AS sales.

```solution
SELECT pgivm.create_immv('product_counts', 'SELECT product, count(*) AS sales FROM sales GROUP BY product');
```

```check
SELECT EXISTS (SELECT FROM pgivm.pg_ivm_immv WHERE immvrelid::text = 'product_counts')
   AND product_counts_match();
```
