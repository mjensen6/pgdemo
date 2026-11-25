---
title: pg_activity
tldr: See what the server is doing right now, find the session that's blocking everyone else, and stop it.
---

```setup
CREATE TABLE accounts (id int PRIMARY KEY, balance int);
INSERT INTO accounts VALUES (1, 100), (2, 50);
-- The check waits a moment for the blocked checkout to finish.
CREATE FUNCTION checkout_done() RETURNS bool LANGUAGE plpgsql AS $$
BEGIN
  FOR i IN 1..20 LOOP
    IF (SELECT balance FROM accounts WHERE id = 1) = 90 THEN RETURN true; END IF;
    PERFORM pg_sleep(0.25);
  END LOOP;
  RETURN false;
END $$;
```

```view
SELECT pid, application_name AS app, state, coalesce(wait_event_type, '') AS waiting_on,
       array_to_string(pg_blocking_pids(pid), ',') AS blocked_by,
       to_char(now() - xact_start, 'MI:SS') AS in_transaction, left(query, 40) AS last_query
FROM pg_stat_activity
WHERE datname = current_database() AND application_name IN ('forgotten-migration', 'checkout', 'slow-report')
ORDER BY pid;
```

```cleanup
SELECT pg_terminate_backend(pid) FROM pg_stat_activity
WHERE datname = 'lesson_pg_activity' AND application_name IN ('forgotten-migration', 'checkout', 'slow-report');
```

When the app suddenly slows down, a common cause is one session in the way of everyone else: a query that has run for an hour, or a transaction someone opened and forgot, still holding locks.

**`pg_stat_activity`** is a view with one row per session on the server. The columns that matter most:

| Column | Meaning |
|---|---|
| `pid` | The session's process ID, used to stop it |
| `state` | `active` (running a query), `idle`, or **`idle in transaction`**: in a transaction that isn't doing anything, but still holds its locks |
| `wait_event_type` | What it's waiting for, if anything. `Lock` means another session's lock |
| `xact_start`, `query_start` | When its transaction and its latest query started |

**`pg_blocking_pids(pid)`** names the sessions blocking a given session. That's how you find the culprit.

**Stopping a session** takes one of two functions:

- **`pg_cancel_backend(pid)`** cancels the query it's running. The session stays connected. It's the gentle option, but it does nothing to a session that's idle in transaction, since there's no query to cancel.
- **`pg_terminate_backend(pid)`** ends the whole session. Its transaction rolls back and its locks are released.

**pg_activity** is a terminal tool that shows all of this live, like `top` for Postgres: the running queries, how long they've taken, what's waiting, and what's blocking. It's the **Top** tab at the bottom of this screen, and `pgdemo top` outside a lesson.

The setting `idle_in_transaction_session_timeout` ends forgotten transactions automatically, and is worth setting in production.

**Where it came from.** `pg_stat_activity` has been in Postgres for over two decades. Postgres 9.6 (2016) added `pg_blocking_pids()` and the wait-event columns, which made finding blockers much easier. pg_activity is a separate open-source tool from Dalibo, a French Postgres company.

**The Live View** shows three pretend sessions that the steps start: their state, what they're waiting on, and who's blocking them.

## Try it

Start a slow report: a query that will take 10 minutes. In the Live View it's `active`, with its query shown.

```sh
PGAPPNAME=slow-report nohup psql -c "SELECT pg_sleep(600) AS report" >/dev/null 2>&1 &
sleep 0.5; echo "The slow report is running."
```

```expect
The slow report is running.
```

Cancel it. The query stops, and the session goes `idle` but stays connected.

```sql
SELECT pg_cancel_backend(pid) FROM pg_stat_activity WHERE application_name = 'slow-report';
```

Now trouble. A migration starts a transaction, updates account 1, and is forgotten: `idle in transaction`, holding the row's lock. Then a checkout tries to update account 1 and has to wait. In the Live View, `checkout` is waiting on `Lock`, and `blocked_by` names the migration's pid.

```sh
# Everything in the background writes to /dev/null, so this step doesn't wait for it.
( { echo "BEGIN; UPDATE accounts SET balance = balance WHERE id = 1;"; sleep 600; } |
  PGAPPNAME=forgotten-migration psql -q ) >/dev/null 2>&1 &
sleep 0.5
PGAPPNAME=checkout nohup psql -c "UPDATE accounts SET balance = balance - 10 WHERE id = 1" >/dev/null 2>&1 &
sleep 0.5; echo "The checkout is waiting."
```

```expect
The checkout is waiting.
```

Find who's blocked, and by whom, the way you would on a real server. Then have a look at the **Top** tab at the bottom of the screen, and click **Lesson** to come back.

```sql
SELECT pid, application_name, state, pg_blocking_pids(pid) AS blocked_by
FROM pg_stat_activity WHERE cardinality(pg_blocking_pids(pid)) > 0;
```

```expect
checkout
active
```

## Challenge

Unblock the checkout. Stop the forgotten migration so its lock is released, and the checkout's update goes through: account 1's balance drops to 90.

Hint: Cancelling won't work on a session that's idle in transaction. Use the function that ends the whole session, on the application_name forgotten-migration, like the cancel step.

```solution
SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE application_name = 'forgotten-migration';
```

```check
SELECT checkout_done()
   AND NOT EXISTS (SELECT FROM pg_stat_activity WHERE application_name = 'forgotten-migration');
```
