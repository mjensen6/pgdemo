---
title: SKIP LOCKED
tldr: Build a job queue on a plain table. Each worker grabs a job no one else holds, without waiting and without extra software.
---

```setup
CREATE EXTENSION pgrowlocks;
CREATE TABLE jobs (id int PRIMARY KEY, task text, done bool NOT NULL DEFAULT false);
INSERT INTO jobs (id, task) VALUES
  (1, 'send welcome email'), (2, 'resize avatar'), (3, 'charge card'),
  (4, 'build report'), (5, 'send receipt'), (6, 'sync calendar');
```

```view
SELECT j.id, j.task, j.done, l.locked_row IS NOT NULL AS locked
FROM jobs j LEFT JOIN pgrowlocks('jobs') l ON l.locked_row = j.ctid
ORDER BY j.id;
```

Many apps need a **job queue**: a list of work, like emails to send, and several **workers** that each take a job, do it, and mark it done. The hard part is making sure two workers never take the same job.

A table can hold the queue. A worker claims a job by locking its row with **`SELECT ... FOR UPDATE`**. The lock lasts until the worker's transaction ends, and anyone else who tries to lock that row has to wait.

That waiting is the problem. If worker A holds the first two jobs, worker B asks for "the first two jobs" too, and it just waits for A to finish. Add more workers and they queue up behind each other, one at a time.

**`SKIP LOCKED`** tells Postgres to skip rows someone else has locked, instead of waiting for them. Worker B gets jobs 3 and 4 straight away. Every worker gets its own jobs, with no waiting and no collisions.

The usual pattern claims a job and marks it in one statement:

```sql
UPDATE jobs SET done = true
WHERE id = (SELECT id FROM jobs WHERE NOT done ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED)
RETURNING id, task;
```

**The limits.** A queue in Postgres is simplest when your data is already there, since a job and the data it changes commit together. For very high volumes, millions of jobs a minute, a dedicated queue like Kafka or RabbitMQ fits better. And remember lesson 1: every finished job is an update that leaves a dead row, so a busy queue table needs autovacuum to keep up.

**Where it came from.** `SKIP LOCKED` arrived in Postgres 9.5 (2016). Oracle had it long before. Earlier Postgres queues used advisory locks or retry loops instead. Today job libraries in many languages build on it, including Oban for Elixir, GoodJob for Ruby, and River for Go, which has its own lesson later.

**The Live View** shows the `jobs` table:

| Column | Meaning |
|---|---|
| `task` | The work to do |
| `done` | Whether a worker finished it |
| `locked` | Whether a worker holds a lock on the row right now, from the `pgrowlocks` extension |

## Try it

Worker A, running in the background, locks the first two jobs and holds them for 5 seconds. Watch `locked` turn on for jobs 1 and 2 in the Live View. Worker B, without `SKIP LOCKED`, waits for them. A 1-second `lock_timeout` makes it give up instead of waiting the full 5 seconds. Then worker B tries again with `SKIP LOCKED` and gets jobs 3 and 4 at once.

```sh
psql -q -c "BEGIN" -c "SELECT id FROM jobs ORDER BY id LIMIT 2 FOR UPDATE" -c "SELECT pg_sleep(5)" -c "COMMIT" >/dev/null &
sleep 0.5
echo "Worker B, waiting:"
psql -c "SET lock_timeout = '1s'" -c "SELECT id FROM jobs ORDER BY id LIMIT 2 FOR UPDATE" 2>&1 | grep -v '^SET'
echo "Worker B, with SKIP LOCKED:"
psql -c "SELECT id FROM jobs ORDER BY id LIMIT 2 FOR UPDATE SKIP LOCKED"
wait
```

```expect
Worker B, waiting:
canceling statement due to lock timeout
Worker B, with SKIP LOCKED:
(2 rows)
```

Now the full pattern from the intro: claim the next job that's not done and mark it done, in one statement. Job 1 turns `done` in the Live View.

```sql
UPDATE jobs SET done = true
WHERE id = (SELECT id FROM jobs WHERE NOT done ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED)
RETURNING id, task;
```

```expect
1 | send welcome email
```

## Challenge

A worker that grabs one job at a time spends a lot of its time asking. Claim the next **3** jobs that aren't done and mark them done, in one statement.

Hint: Take the last step, raise the LIMIT, and change id = to id IN, since the subquery now returns several rows.

```solution
UPDATE jobs SET done = true
WHERE id IN (SELECT id FROM jobs WHERE NOT done ORDER BY id LIMIT 3 FOR UPDATE SKIP LOCKED)
RETURNING id, task;
```

```check
SELECT array_agg(id ORDER BY id) = '{1,2,3,4}' FROM jobs WHERE done;
```
