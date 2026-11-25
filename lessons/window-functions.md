---
title: Window functions
tldr: Rank, running totals, and comparisons with the previous row, computed across related rows without collapsing them the way GROUP BY does.
---

```setup
CREATE TABLE sales (region text, rep text, month date, amount int);
INSERT INTO sales VALUES
  ('North', 'Ada', '2026-01-01', 120), ('North', 'Ada', '2026-02-01', 90),  ('North', 'Ada', '2026-03-01', 150),
  ('North', 'Bo',  '2026-01-01', 80),  ('North', 'Bo',  '2026-02-01', 110), ('North', 'Bo',  '2026-03-01', 60),
  ('North', 'Cy',  '2026-01-01', 40),  ('North', 'Cy',  '2026-02-01', 70),  ('North', 'Cy',  '2026-03-01', 50),
  ('South', 'Di',  '2026-01-01', 200), ('South', 'Di',  '2026-02-01', 30),  ('South', 'Di',  '2026-03-01', 90),
  ('South', 'Ed',  '2026-01-01', 60),  ('South', 'Ed',  '2026-02-01', 140), ('South', 'Ed',  '2026-03-01', 130),
  ('South', 'Flo', '2026-01-01', 50),  ('South', 'Flo', '2026-02-01', 40),  ('South', 'Flo', '2026-03-01', 20);
CREATE TABLE top_reps (region text, rep text, total bigint);
```

```view
SELECT region, rep, total FROM top_reps ORDER BY region, total DESC;
```

`GROUP BY` answers questions like "total sales per region", but it collapses each group into one row, so the individual rows are gone. Many questions need both: each rep's sales **and** their rank in the region, each month **and** the running total so far.

A **window function** computes a value across a set of related rows, its **window**, and puts the answer on every row, without collapsing anything. You write it with **`OVER (...)`**:

```sql
SELECT rep, amount, sum(amount) OVER (PARTITION BY region) AS region_total FROM sales;
```

Inside `OVER`:

- **`PARTITION BY`** splits the rows into groups, like `GROUP BY` but without merging them. Each row only sees its own group.
- **`ORDER BY`** sorts the rows within each group. With it, `sum()` becomes a **running total**, adding up the rows from the start of the group to the current one.

Some functions only make sense over a window:

| Function | Gives |
|---|---|
| `row_number()` | 1, 2, 3, ... in order |
| `rank()` | Position in order, with ties sharing a rank and leaving a gap after |
| `lag(x)` | `x` from the previous row, so you can compare with it |

And any aggregate, like `sum`, `avg`, or `count`, works as a window function with `OVER`.

**One rule to know.** Window functions run after `WHERE`, so you can't filter on one in the same query's `WHERE`. Put the query in a subquery and filter outside it, as the second step shows.

**Where it came from.** Window functions are in the SQL standard since SQL:2003, and arrived in Postgres 8.4 (2009). Before that, you needed self-joins or correlated subqueries, which were slow and hard to read.

**The Live View** shows table `top_reps`, which you'll fill in the challenge.

## Try it

Each sale next to its region's total. Every row stays, and `region_total` repeats within each region.

```sql
SELECT region, rep, month, amount, sum(amount) OVER (PARTITION BY region) AS region_total
FROM sales WHERE month = '2026-01-01' ORDER BY region, rep;
```

Rank reps by their total within their region. The inner query sums per rep with `GROUP BY`. The window then ranks those totals within each region. The outer query keeps only first place, since the inner query can't filter on `rank` itself.

```sql
SELECT * FROM (
  SELECT region, rep, sum(amount) AS total,
         rank() OVER (PARTITION BY region ORDER BY sum(amount) DESC) AS rank
  FROM sales GROUP BY region, rep
) ranked
WHERE rank = 1;
```

Ada's month by month: a running total with `ORDER BY` in the window, and the change from last month with `lag`.

```sql
SELECT month, amount,
       sum(amount) OVER (ORDER BY month) AS running_total,
       amount - lag(amount) OVER (ORDER BY month) AS change
FROM sales WHERE rep = 'Ada' ORDER BY month;
```

## Challenge

Fill `top_reps` (region, rep, total) with the **top 2** reps in each region by total sales. Watch the Live View.

Hint: INSERT INTO top_reps, then the second step's query with a different WHERE on rank. Select only region, rep, and total.

```solution
INSERT INTO top_reps
SELECT region, rep, total FROM (
  SELECT region, rep, sum(amount) AS total,
         rank() OVER (PARTITION BY region ORDER BY sum(amount) DESC) AS rank
  FROM sales GROUP BY region, rep
) ranked
WHERE rank <= 2;
```

```check
-- North: Ada 360, Bo 250. South: Di 320, Ed 330.
SELECT count(*) = 4 AND NOT EXISTS (
  SELECT region, rep, total FROM top_reps
  EXCEPT SELECT * FROM (VALUES ('North', 'Ada', 360), ('North', 'Bo', 250), ('South', 'Ed', 330), ('South', 'Di', 320)) v)
FROM top_reps;
```
