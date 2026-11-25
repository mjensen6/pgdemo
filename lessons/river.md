---
title: River
tldr: A real Go job queue that keeps its jobs in a Postgres table, so enqueueing a job can be part of the same transaction as your data.
---

```setup
\! river-demo -d lesson_river migrate >/dev/null
CREATE TABLE users (id serial PRIMARY KEY, email text);
```

```view
SELECT id, kind, args->>'to' AS sending_to, state, attempt,
       CASE WHEN errors IS NULL THEN '' ELSE errors[array_length(errors, 1)]->>'error' END AS last_error
FROM river_job ORDER BY id DESC LIMIT 10;
```

**River** is a job queue library for Go. Apps use it for work that shouldn't happen inside a web request: sending email, resizing images, calling slow APIs. The app **enqueues** a job, and **workers** pick it up and run it in the background.

Unlike queues that need their own server, such as Redis or RabbitMQ, River keeps every job as a row in a Postgres table, `river_job`. It builds on three things from earlier lessons:

- **SKIP LOCKED** (lesson 19): workers claim jobs without ever taking the same one twice.
- **LISTEN/NOTIFY** (lesson 20): a new job wakes the workers at once, instead of them polling.
- **Transactions**: a job is just a row, so you can enqueue it in the **same transaction** as the data it's about.

That last point is the big one. With a separate queue server, "save the new user" and "queue their welcome email" are two systems. If the save rolls back after the email was queued, someone gets welcomed to an account that doesn't exist. If the queue is down after the save commits, the email is lost. In Postgres, both commit together or neither does.

Each job has a **state**:

| State | Meaning |
|---|---|
| `available` | Waiting for a worker |
| `running` | A worker has it |
| `retryable` | It failed, and will be tried again later |
| `completed` | Done |
| `discarded` | It failed too many times, `max_attempts`, and was given up on |

When a job fails, River records the error in the job's row and retries it later, waiting longer after each failure.

**The demo app.** This course includes a small Go program, `river-demo`, built with River. It runs here with `river-demo -d lesson_river` followed by `migrate` (create River's tables, already done), `enqueue N` (add N "send email" jobs), or `work SECONDS` (run 3 workers for a while). Jobs to an address containing `flaky` fail on their first attempt, to show a retry.

**Where it came from.** River is open source, created by Brandur Leach and Blake Gentry and first released in 2023. It joins a line of Postgres-backed job queues in other languages, such as Oban for Elixir and GoodJob for Ruby. Their argument: if your data is already in Postgres, your jobs can be too.

**The Live View** shows the newest jobs in `river_job`, with their state, attempts, and last error.

## Try it

Enqueue 5 email jobs with the demo app. They appear in the Live View as `available`. Job 2 goes to the flaky address.

```sh
river-demo -d lesson_river enqueue 5
```

```expect
enqueued job 1
enqueued job 2: send email to flaky@example.com
enqueued job 5
```

Each job is an ordinary row. Its arguments are JSON, and `kind` tells River which worker runs it.

```sql
SELECT id, kind, args, state, max_attempts FROM river_job ORDER BY id;
```

```expect
send_email
available
(5 rows)
```

Run 3 workers for 6 seconds. The jobs go to `completed`, except job 2, which fails first and shows `retryable` with its error. A moment later River retries it, and it completes on attempt 2.

```sh
river-demo -d lesson_river work 6
```

```expect
job 2 to flaky@example.com failed on attempt 1, River will retry it
job 2 sent email to flaky@example.com (attempt 2)
```

Enqueue from SQL, inside your own transaction. Insert a user and their welcome-email job together, then roll back. Neither the user nor the job exists afterwards. No email for an account that was never created.

```sql
BEGIN;
INSERT INTO users (email) VALUES ('grace@example.com');
INSERT INTO river_job (kind, args) VALUES ('send_email', '{"to": "grace@example.com"}');
ROLLBACK;
SELECT count(*) AS jobs_for_grace FROM river_job WHERE args->>'to' = 'grace@example.com';
```

```expect
ROLLBACK
jobs_for_grace
```

## Challenge

Sign up a new user, `ada@example.com`, and enqueue their welcome email **in the same transaction**, and commit it. Then run the workers from the SQL pane so the email is sent.

Hint: The last step, ending in COMMIT instead of ROLLBACK. Then run the work command from the third step, with \! in front.

```solution
BEGIN;
INSERT INTO users (email) VALUES ('ada@example.com');
INSERT INTO river_job (kind, args) VALUES ('send_email', '{"to": "ada@example.com"}');
COMMIT;
\! river-demo -d lesson_river work 3
```

```check
SELECT EXISTS (SELECT FROM users WHERE email = 'ada@example.com')
   AND EXISTS (SELECT FROM river_job WHERE args->>'to' = 'ada@example.com' AND state = 'completed');
```
