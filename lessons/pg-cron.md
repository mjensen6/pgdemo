---
title: pg_cron
tldr: Schedule SQL to run by itself, every few seconds or every night at 3, inside the database, with no cron server to manage.
---

```setup
CREATE EXTENSION dblink;
CREATE TABLE ticks (at timestamptz DEFAULT now());
\c postgres
CREATE EXTENSION IF NOT EXISTS pg_cron;
```

```view
SELECT j.jobname, j.schedule, j.command, (SELECT count(*) FROM ticks) AS ticks_so_far
FROM dblink('dbname=postgres', $$SELECT jobname, schedule, command FROM cron.job WHERE database = 'lesson_pg_cron'$$)
  AS j(jobname text, schedule text, command text)
UNION ALL
SELECT '(no jobs)', '', '', (SELECT count(*) FROM ticks)
WHERE NOT EXISTS (SELECT FROM dblink('dbname=postgres', $$SELECT 1 FROM cron.job WHERE database = 'lesson_pg_cron'$$) AS j(x int))
ORDER BY 1;
```

```cleanup
\c postgres
SELECT cron.unschedule(jobid) FROM cron.job WHERE database = 'lesson_pg_cron';
```

Databases need regular chores: delete old sessions every night, refresh a summary every hour, run pg_partman's maintenance (lesson 27). Traditionally a **cron** job on some server runs a script that connects and does it. That server is one more thing to keep running and to remember when you move the database.

**pg_cron** runs the schedule inside Postgres itself. A job is a schedule, a SQL command, and the database to run it in:

```sql
SELECT cron.schedule_in_database('nightly-cleanup', '0 3 * * *', 'DELETE FROM sessions WHERE expires < now()', 'mydb');
```

The schedule uses classic **cron syntax**, five fields: minute, hour, day of month, month, day of week. A `*` means "every":

| Schedule | Runs |
|---|---|
| `'0 3 * * *'` | Every day at 03:00 |
| `'*/15 * * * *'` | Every 15 minutes |
| `'0 9 * * 1'` | Every Monday at 09:00 |
| `'5 seconds'` | Every 5 seconds, a pg_cron extra |

Times are in UTC. `cron.unschedule('name')` removes a job. The tables `cron.job` and `cron.job_run_details` show what's scheduled and how each run went.

**One database holds the schedule.** pg_cron is loaded at server start and keeps its jobs in one database, set by `cron.database_name`, which is `postgres` here. So you schedule from that database and name the database the job should run in. Switch databases in psql with **`\c name`**.

**The catch.** A job that fails doesn't tell anyone unless you look at `cron.job_run_details`, so check it, and alert on failures in production. pg_cron also runs jobs only while the server is up. It doesn't catch up on runs it missed.

**Where it came from.** pg_cron is a third-party extension, released by Citus Data in 2016. It isn't part of core Postgres, but most hosted Postgres services offer it, and it's how many teams schedule maintenance today.

This lesson shares the whole server, since the jobs live in the `postgres` database. Leaving the lesson removes its jobs.

**The Live View** shows the jobs scheduled for this playground, and how many rows they've added to table `ticks`.

## Try it

Switch to the `postgres` database, schedule a job that adds a row to `ticks` every second in this playground, and switch back. Watch `ticks_so_far` climb in the Live View.

```sql
\c postgres
SELECT cron.schedule_in_database('tick', '1 second', 'INSERT INTO ticks DEFAULT VALUES', 'lesson_pg_cron');
\c lesson_pg_cron
```

Wait 3 seconds, then count. A job that's scheduled isn't proof it runs, so always check that it really did.

```sql
SELECT pg_sleep(3);
SELECT count(*) AS ticks FROM ticks;
```

Look at the last few runs: when they started, and whether they succeeded. Then remove the job, so it stops adding rows.

```sql
\c postgres
SELECT status, return_message, start_time::time(0) AS started
FROM cron.job_run_details ORDER BY start_time DESC LIMIT 3;
SELECT cron.unschedule('tick');
\c lesson_pg_cron
```

## Challenge

Schedule a job named `cleanup` that runs every day at 03:00 in this playground and deletes the rows of `ticks` older than one day.

Hint: Same steps as the first one: \c postgres, then cron.schedule_in_database with a schedule from the intro's table. The command is DELETE FROM ticks WHERE at < now() - interval '1 day'.

```solution
\c postgres
SELECT cron.schedule_in_database('cleanup', '0 3 * * *', $$DELETE FROM ticks WHERE at < now() - interval '1 day'$$, 'lesson_pg_cron');
\c lesson_pg_cron
```

```check
-- The tick job really ran during the steps, and the cleanup job is scheduled.
SELECT (SELECT count(*) FROM ticks) > 0
   AND EXISTS (SELECT FROM dblink('dbname=postgres',
                 $$SELECT 1 FROM cron.job WHERE jobname = 'cleanup' AND schedule = '0 3 * * *'
                   AND database = 'lesson_pg_cron' AND command ILIKE '%delete%ticks%'$$) AS j(x int));
```
