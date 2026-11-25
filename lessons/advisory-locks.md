---
title: Advisory locks
tldr: Lock on a number your app chooses, to make sure only one worker runs a job at a time, with no table to lock.
---

```setup
CREATE TABLE reports (id serial PRIMARY KEY, made_by text, at timestamptz DEFAULT now());
```

```view
SELECT objid AS lock_key, pid AS held_by_pid,
       (SELECT coalesce(nullif(application_name, ''), '(none)') FROM pg_stat_activity a WHERE a.pid = l.pid) AS client,
       CASE WHEN granted THEN 'held' ELSE 'waiting' END AS status
FROM pg_locks l
WHERE locktype = 'advisory' AND database = (SELECT oid FROM pg_database WHERE datname = current_database())
ORDER BY objid, granted DESC;
```

Sometimes a lock should protect a **task**, not a row. Only one server should run the nightly report, even though all five run the same schedule. Only one deploy should run database migrations at a time. There's no particular row to lock with `FOR UPDATE`.

An **advisory lock** is a lock on a number that you choose. Postgres doesn't know what the number means. It's "advisory" because your app gives it the meaning, say "lock 42 means the nightly report", and everyone who cares agrees to ask for lock 42 first.

| Function | Does |
|---|---|
| `pg_advisory_lock(42)` | Take lock 42, waiting if someone else holds it |
| `pg_try_advisory_lock(42)` | Take it if it's free. Returns `true` or `false`, and never waits |
| `pg_advisory_unlock(42)` | Release it |
| `pg_advisory_xact_lock(42)` | Take it until the current transaction ends. No unlock needed |

**Session or transaction.** The plain functions hold the lock for your whole **session**, until you unlock it or disconnect. The `_xact_` versions release it automatically at `COMMIT` or `ROLLBACK`, which is harder to get wrong. Either way, if a worker crashes, its connection closes and Postgres releases its locks, so a crash can't leave a lock stuck.

`pg_try_advisory_lock` is the usual pattern for "only one of us does this": each worker tries, and those that get `false` skip the job.

**A catch with poolers.** With PgBouncer in transaction mode (lesson 43), consecutive transactions can run on different connections, so a session-level lock can end up held by a connection you no longer have. Use the `_xact_` versions there.

**Where it came from.** Advisory locks arrived in Postgres 8.2 (2006), replacing an older contrib module called `userlock` that did the same thing as an add-on. Many tools rely on them today: Rails, golang-migrate, and others take one before running migrations, so two deploys can't migrate at once.

**The Live View** lists the advisory locks in this playground:

| Column | Meaning |
|---|---|
| `lock_key` | The number |
| `held_by_pid`, `client` | The session that holds or wants it. `psql` is the lesson's own session, `pgcli` your SQL pane |
| `status` | `held`, or `waiting` for someone else to let go |

## Try it

Two workers race for the nightly report. Worker A takes lock 42 and spends 4 seconds on the report. Worker B tries a moment later, gets `false`, and skips. Watch lock 42 in the Live View while A holds it.

```sh
PGAPPNAME=worker-a psql -q -c "SELECT pg_advisory_lock(42)" \
  -c "INSERT INTO reports (made_by) VALUES ('worker A')" -c "SELECT pg_sleep(4)" >/dev/null &
sleep 0.5
PGAPPNAME=worker-b psql -Atc "SELECT CASE WHEN pg_try_advisory_lock(42)
  THEN 'Worker B got the lock' ELSE 'Worker B: someone else is making the report, skipping' END"
wait
echo "Worker A finished, and disconnecting released its lock."
```

```expect
Worker B: someone else is making the report, skipping
Worker A finished
```

Take a session-level lock in the lesson's own session. It shows in the Live View as held by `psql`, and stays until you unlock it.

```sql
SELECT pg_advisory_lock(7);
```

```expect
pg_advisory_lock
```

Release it, and it's gone from the Live View.

```sql
SELECT pg_advisory_unlock(7);
```

```expect
t
```

A transaction-level lock needs no unlock. It appears during the transaction and disappears at `COMMIT`.

```sql
BEGIN;
SELECT pg_advisory_xact_lock(99);
SELECT objid FROM pg_locks WHERE locktype = 'advisory';
COMMIT;
```

```expect
99
COMMIT
```

## Challenge

Lock **2026** stands for this year's archive job. Take it in your SQL pane session and keep holding it, so any other worker that tries for it skips.

Hint: The second step took a session-level lock that stayed until it was unlocked. Use the same function with 2026, and don't unlock it.

```solution
SELECT pg_advisory_lock(2026);
```

```check
SELECT EXISTS (SELECT FROM pg_locks
               WHERE locktype = 'advisory' AND objid = 2026 AND granted
                 AND database = (SELECT oid FROM pg_database WHERE datname = current_database()));
```
