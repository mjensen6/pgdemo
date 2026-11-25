---
title: Autovacuum, and the scores new in Postgres 19
tldr: Autovacuum cleans dead rows for you once a table passes a threshold. Postgres 19 shows each table's score and cleans the neediest first.
---

```setup
CREATE TABLE busy (id int) WITH (autovacuum_enabled = false);
CREATE TABLE calm (id int) WITH (autovacuum_enabled = false);
INSERT INTO busy SELECT generate_series(1, 50000);
INSERT INTO calm SELECT generate_series(1, 50000);
VACUUM ANALYZE busy, calm;
DELETE FROM busy WHERE id % 2 = 0;
DELETE FROM calm WHERE id % 100 = 0;
```

```view
SELECT s.relname AS "table", t.n_live_tup AS live, t.n_dead_tup AS dead,
       round(s.vacuum_score::numeric, 2) AS vacuum_score, s.do_vacuum AS due,
       to_char(t.last_autovacuum, 'HH24:MI:SS') AS last_autovacuum
FROM pg_stat_autovacuum_scores s JOIN pg_stat_user_tables t ON t.relid = s.relid
WHERE s.relname IN ('busy', 'calm') ORDER BY 1;
```

Lesson 1 showed that updates and deletes leave **dead rows**, and that VACUUM clears them for reuse. **Autovacuum** is the background process that runs VACUUM for you. It also runs ANALYZE, which refreshes the statistics the planner uses. You rarely notice it, which is the idea.

**When does a table get vacuumed?** Every minute, autovacuum checks each table against a threshold:

```
dead rows  >  autovacuum_vacuum_threshold  +  autovacuum_vacuum_scale_factor × rows
              (default 50)                    (default 0.2, so 20%)
```

With the defaults, a table of 50,000 rows is vacuumed once it has more than 10,050 dead rows. You can change both numbers for one table:

```sql
ALTER TABLE calm SET (autovacuum_vacuum_scale_factor = 0.01);
```

The 20% default suits most tables, but it's far too lazy for a huge one. A billion-row table would collect 200 million dead rows first. Big, busy tables often get a scale factor of 1% or less.

**New in Postgres 19: scores.** The view `pg_stat_autovacuum_scores` shows how close each table is to its threshold:

| Column | Meaning |
|---|---|
| `vacuum_score` | Dead rows divided by the threshold. 1 or more means vacuum is due |
| `score` | The highest of the table's scores, counting analyze and others |
| `do_vacuum` | Whether autovacuum would vacuum it now. False if it's turned off for the table |

Postgres 19 also uses the scores to choose **which table goes first**, the highest score first, so the most urgent table no longer waits behind others. And autovacuum can now use **parallel workers** to clean a table's indexes faster. It's off by default and set with `autovacuum_max_parallel_workers`.

**Where it came from.** Autovacuum started as an add-on, the `pg_autovacuum` contrib module in Postgres 7.4 (2003). So many people ran it that it moved into the server in 8.1 (2005), and has been on by default since 8.3 (2008). The scores, ordering by score, and parallel workers are new in Postgres 19 (2026).

**The Live View** watches two tables of 50,000 rows each, with autovacuum turned off for both:

| Column | Meaning |
|---|---|
| `live`, `dead` | Live and dead rows |
| `vacuum_score`, `due` | From the new view, explained above |
| `last_autovacuum` | When autovacuum last cleaned the table |

## Try it

`busy` had half its rows deleted, 25,000 dead rows, so its `vacuum_score` is about 2.5, well past 1. But `due` is false, because autovacuum is off for it. `calm` lost only 1%, 500 rows, and scores about 0.05.

```sql
SELECT relname, round(vacuum_score::numeric, 2) AS vacuum_score, do_vacuum
FROM pg_stat_autovacuum_scores WHERE relname IN ('busy', 'calm') ORDER BY 1;
```

```expect
busy    |         2.49 | f
calm    |         0.05 | f
```

Turn autovacuum on for `busy`. `due` becomes true. Autovacuum checks once a minute, so within a minute, watch `dead` drop to 0 and `last_autovacuum` fill in. You can carry on meanwhile.

```sql
ALTER TABLE busy SET (autovacuum_enabled = true);
```

```expect
ALTER TABLE
```

## Challenge

`calm` scores only about 0.05 with the default 20% scale factor, so it isn't due yet. Change its scale factor so its 500 dead rows count as due. The Live View should show its `vacuum_score` at 1 or more.

Hint: The threshold is 50 + scale factor × 50,000. Pick a scale factor that brings it under 500, and set it with ALTER TABLE calm SET (...), like the intro shows.

```solution
ALTER TABLE calm SET (autovacuum_vacuum_scale_factor = 0.005);
```

```check
SELECT vacuum_score >= 1 FROM pg_stat_autovacuum_scores WHERE relname = 'calm';
```
