---
title: pg_plan_advice, new in Postgres 19
tldr: Write down a query plan as advice, then hand it back so the planner sticks to it. A safety valve for the rare plan that goes wrong.
---

```setup
ALTER DATABASE lesson_plan_advice SET session_preload_libraries = 'pg_plan_advice';
CREATE TABLE a AS SELECT i AS id FROM generate_series(1, 10000) i;
CREATE TABLE b AS SELECT i AS id FROM generate_series(1, 10000) i;
CREATE INDEX b_id ON b (id);
ANALYZE a, b;
-- The check uses this to see the plan a new session gets.
CREATE FUNCTION plan(q text) RETURNS SETOF text LANGUAGE plpgsql AS $$
BEGIN RETURN QUERY EXECUTE 'EXPLAIN (COSTS OFF) ' || q; END $$;
```

```view
SELECT coalesce((SELECT array_to_string(setconfig, E'\n') FROM pg_db_role_setting
                 WHERE setrole = 0 AND setdatabase = (SELECT oid FROM pg_database WHERE datname = current_database())),
                '(none)') AS playground_defaults;
```

For every query, the **planner** chooses a plan: which table to read first, whether to use an index, and how to join. It picks the plan its estimates say is cheapest, and it's right almost all the time.

Very occasionally it's wrong, often after a data change makes an estimate go stale, and a query that took milliseconds suddenly takes minutes. You need a way to say "use the plan that worked" while you fix the cause.

**pg_plan_advice** gives you that. `EXPLAIN (PLAN_ADVICE)` writes the chosen plan as **advice**, a short description like:

```
JOIN_ORDER(a b)  NESTED_LOOP_PLAIN(b)  SEQ_SCAN(a)  INDEX_ONLY_SCAN(b public.b_id)
```

Put advice in the `pg_plan_advice.advice` setting, and the planner follows it. You can give a whole plan, or only the part you care about, like `HASH_JOIN(b)`. EXPLAIN then marks each piece `matched`, or `not matched` if it doesn't fit the query.

**Advice, not orders.** It's advice because it can't make the planner do something impossible, such as use an index that doesn't exist. Use it as a short-term fix. Better statistics, an index, or a rewritten query are the lasting fixes. A pinned plan can't adapt when your data changes.

**Where it came from.** `pg_plan_advice` is a contrib module new in Postgres 19 (2026), written by Robert Haas. For over a decade, people who needed this used `pg_hint_plan`, a third-party extension that reads hints from comments in the query. `pg_plan_advice` isn't a copy of it. It builds the round trip from plan to advice and back into Postgres itself, and gives tools like `pg_hint_plan` a supported foundation.

This playground loads the module for every session, through `session_preload_libraries`. Normally you'd run `LOAD 'pg_plan_advice';` first.

**The Live View** shows `playground_defaults`: settings saved for new sessions in this playground with `ALTER DATABASE`, which you met in the async I/O lesson.

## Try it

Ask for the plan as advice. For a few rows of `a`, the planner chooses a nested loop that looks up each one in `b`'s index. The `Generated Plan Advice` section describes exactly that.

```sql
EXPLAIN (COSTS OFF, PLAN_ADVICE) SELECT * FROM a JOIN b ON a.id = b.id WHERE a.id < 10;
```

Now advise a hash join with a full scan of `b`. The plan changes, and both pieces say `matched`.

```sql
SET pg_plan_advice.advice = 'HASH_JOIN(b) SEQ_SCAN(b)';
EXPLAIN (COSTS OFF) SELECT * FROM a JOIN b ON a.id = b.id WHERE a.id < 10;
```

Advice that doesn't fit the query is ignored, and EXPLAIN says so. There's no table `x` here, so this is `not matched`, and the planner goes back to its own choice.

```sql
SET pg_plan_advice.advice = 'HASH_JOIN(x)';
EXPLAIN (COSTS OFF) SELECT * FROM a JOIN b ON a.id = b.id WHERE a.id < 10;
RESET pg_plan_advice.advice;
```

## Challenge

For `a.id < 5000`, the planner picks a hash join. Make every **new** session in this playground use a plain nested loop into `b` for that join instead.

Hint: The advice for that appeared in the first step's output. Save it the way the async I/O lesson saved a setting: ALTER DATABASE lesson_plan_advice SET ...

```solution
ALTER DATABASE lesson_plan_advice SET pg_plan_advice.advice = 'NESTED_LOOP_PLAIN(b)';
```

```check
-- The check runs in a new session, so it gets the playground's saved defaults.
SELECT bool_or(line LIKE 'Nested Loop%')
FROM plan('SELECT * FROM a JOIN b ON a.id = b.id WHERE a.id < 5000') AS line;
```
