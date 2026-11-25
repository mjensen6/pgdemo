---
title: HyperLogLog
tldr: Count unique visitors approximately, with about 2% error, in about a kilobyte, and combine days into weeks, which exact counts can't do.
---

```setup
CREATE EXTENSION hll;
CREATE TABLE visits AS
  SELECT date '2026-01-01' + (i - 1) / 40000 AS day, (i::bigint * 7919) % 50000 AS user_id
  FROM generate_series(1, 320000) i;
ANALYZE visits;
CREATE TABLE daily_uniques (day date PRIMARY KEY, users hll);
```

```view
SELECT day::text AS day, round(hll_cardinality(users)) AS estimated_unique_users, pg_column_size(users) AS bytes
FROM daily_uniques
UNION ALL
SELECT 'all days, combined', round(hll_cardinality(hll_union_agg(users))), pg_column_size(hll_union_agg(users))
FROM daily_uniques HAVING count(*) > 0
ORDER BY 1;
```

"How many **unique** visitors did we have?" sounds simple. `SELECT count(DISTINCT user_id)` answers it, but it has to keep every user ID it has seen, which gets slow and memory-hungry at billions of visits. And it doesn't **combine**: if Monday had 40,000 unique visitors and Tuesday 40,000, the two days together could have anywhere from 40,000 to 80,000. You can't add them up. You have to go back to the raw visits.

**HyperLogLog** is an algorithm that **estimates** the number of distinct values. It hashes each value and keeps a small summary based on the hashes. That summary, called an **hll**, takes about 1 kilobyte no matter how many values went in, and its estimate is usually within about 2%.

The extension `hll` adds an `hll` column type and functions:

| Function | Does |
|---|---|
| `hll_hash_bigint(x)`, `hll_hash_text(x)` | Hash a value, ready to add |
| `hll_add_agg(hash)` | Build an hll from many rows, like an aggregate |
| `hll_cardinality(h)` | The estimated number of distinct values |
| `hll_union_agg(h)` | Combine several hlls into one, like an aggregate |

**Union is the superpower.** Combining two hlls gives the hll of everything in both, so the estimate is for unique visitors across both days, with no double counting. So you keep one tiny hll per day in a **rollup table**, and answer "unique visitors this week, this month, in March" by combining the days, without touching the raw data.

**The trade-off** is that it's an estimate. Fine for analytics dashboards. Not for billing, or anything that must be exact.

**Where it came from.** The HyperLogLog algorithm was published by Philippe Flajolet and colleagues in 2007. The `postgresql-hll` extension came from Aggregate Knowledge in 2013, and is now maintained by Citus Data. Big analytics systems such as BigQuery and Redis offer HyperLogLog counts too.

**The Live View** shows the rollup table `daily_uniques`: each day's estimate, its size in bytes, and all the days combined. The data covers 8 days, 2026-01-01 to 2026-01-08.

## Try it

The exact count over all 8 days, and the HyperLogLog estimate from a single hll. They're within a few percent of each other.

```sql
SELECT count(DISTINCT user_id) AS exact,
       round(hll_cardinality(hll_add_agg(hll_hash_bigint(user_id)))) AS estimate,
       pg_column_size(hll_add_agg(hll_hash_bigint(user_id))) AS hll_bytes
FROM visits;
```

```expect
50000
48881
1287
```

Build the rollup for the first 7 days: one hll per day. Watch the days appear in the Live View, each about a kilobyte.

```sql
INSERT INTO daily_uniques
SELECT day, hll_add_agg(hll_hash_bigint(user_id)) FROM visits WHERE day <= '2026-01-07' GROUP BY day;
```

```expect
INSERT 0 7
```

Now the problem with exact counts. Adding up each day's unique visitors counts the same people again and again. Combining the hlls doesn't: it estimates the real number of unique visitors for the week.

```sql
SELECT (SELECT sum(c) FROM (SELECT count(DISTINCT user_id) AS c FROM visits WHERE day <= '2026-01-07' GROUP BY day) d) AS sum_of_daily_counts,
       (SELECT round(hll_cardinality(hll_union_agg(users))) FROM daily_uniques) AS combined_hll,
       (SELECT count(DISTINCT user_id) FROM visits WHERE day <= '2026-01-07') AS exact;
```

```expect
280000
48881
50000
```

## Challenge

The rollup is missing the last day, 2026-01-08. Add its hll to `daily_uniques`, so the Live View's combined estimate covers all 8 days.

Hint: The rollup step's INSERT, for that one day.

```solution
INSERT INTO daily_uniques
SELECT day, hll_add_agg(hll_hash_bigint(user_id)) FROM visits WHERE day = '2026-01-08' GROUP BY day;
```

```check
-- The day is there, and its estimate is within 5% of the exact count.
SELECT count(*) = 8 AND bool_or(day = '2026-01-08' AND abs(hll_cardinality(users) -
         (SELECT count(DISTINCT user_id) FROM visits WHERE day = '2026-01-08')) <
         0.05 * (SELECT count(DISTINCT user_id) FROM visits WHERE day = '2026-01-08'))
FROM daily_uniques;
```
