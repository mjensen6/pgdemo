# pgdemo

Learn modern PostgreSQL 19 in your terminal, one minute at a time. Each lesson runs a few real commands, shows you what changed, and ends with a small challenge you solve yourself.

All you need is Docker. Everything else, including Postgres, runs inside the course's own container and never touches anything already on your machine.

## Start

```bash
git clone <this repo> pgdemo
cd pgdemo
./pgdemo
```

The first run checks Docker, builds the course (a few minutes, once), and opens lesson 1. Run `./pgdemo` again any time to pick up where you left off.

Tested on Linux. If you try it on macOS or anywhere else, please open an issue and say how it went.

## The screen

```
┌─ Lesson 1: MVCC and bloat ─┬─ SQL ─────────────────────┐
│ An UPDATE leaves the old   │ lesson=# \dt              │
│ row behind as a dead row.  │                           │
│                            ├─ Live View ───────────────┤
│ UPDATE t SET note = ...    │ size   | rows  | dead     │
│                            │ 912 kB | 10000 | 10000    │
│ Enter next  b back  c check  h hint  s skip            │
│ n next lesson  ? help  q quit                          │
└────────────────────────────┴───────────────────────────┘
 Lesson | Browse | Top
```

- **Lesson**: what is happening and why. Press Enter to run each step.
- **SQL**: your own prompt, already connected to the lesson's playground. Click it to type.
- **Live View**: the numbers the lesson wants you to watch, refreshed every second.

Click to switch panes and tabs. Press `?` for every key. Each lesson has its own playground database, so break whatever you like. `pgdemo 1 reset` rebuilds it.

## Commands

```
pgdemo                  Start, or resume where you left off
pgdemo list             All lessons by Section, with progress
pgdemo 5                Open lesson 5 (a name works too: pgdemo repack)
pgdemo 5 tldr           What it is and why it matters. Runs nothing
pgdemo 5 info           The full written lesson. Runs nothing
pgdemo 5 run            Run every step non-stop
pgdemo 5 reset          Wipe the playground and return to step 1
pgdemo 5 check          Verify the lesson was completed
pgdemo 5 hint           Hint for the challenge
pgdemo 5 solution       Show the answer
pgdemo check            Re-verify every lesson you have opened
pgdemo psql             Open plain psql
pgdemo stop             Stop the database
pgdemo help             All commands
```

Your progress is in the `progress` table of the `pgdemo` database, so you can query it with SQL.

## Lessons

<!-- lessons:start (generated from lessons/course.yaml) -->

**1. Core Concepts**

1. [MVCC and bloat](lessons/mvcc-bloat.md): Updates leave dead rows behind. VACUUM reuses the space but rarely shrinks the file.
2. [Transactional DDL](lessons/transactional-ddl.md): Schema changes can be rolled back like any other change, so a failed migration leaves nothing half done.

**2. Modern SQL**

3. `window-functions` _(coming soon)_
4. `recursive-queries` _(coming soon)_
5. `merge` _(coming soon)_
6. `on-conflict-do-select` _(coming soon)_
7. `generated-columns` _(coming soon)_
8. `uuid` _(coming soon)_
9. `materialized-views` _(coming soon)_

**3. Documents and Search**

10. `jsonb` _(coming soon)_
11. `json-table` _(coming soon)_
12. `full-text-search` _(coming soon)_

**4. Indexes and Performance**

13. `index-toolbox` _(coming soon)_
14. `skip-scan` _(coming soon)_
15. `async-io` _(coming soon)_
16. `pg-stat-statements` _(coming soon)_
17. `plan-advice` _(coming soon)_
18. `hypopg` _(coming soon)_

**5. Queues and Events**

19. `skip-locked` _(coming soon)_
20. `listen-notify` _(coming soon)_
21. `advisory-locks` _(coming soon)_
22. `river` _(coming soon)_

**6. Security and Testing**

23. `row-level-security` _(coming soon)_
24. `pgaudit` _(coming soon)_
25. `pgtap` _(coming soon)_

**7. Data at Scale**

26. `partitioning` _(coming soon)_
27. `pg-partman` _(coming soon)_
28. `pg-ivm` _(coming soon)_
29. `hll` _(coming soon)_

**8. Keeping It Healthy**

30. `pg-activity` _(coming soon)_
31. [REPACK, new in Postgres 19](lessons/repack.md): REPACK rebuilds a bloated table to its real size. With CONCURRENTLY, the app keeps writing while it runs.
32. `pg-repack` _(coming soon)_
33. `pg-squeeze` _(coming soon)_
34. `autovacuum` _(coming soon)_

**9. Beyond Tables**

35. `pgvector` _(coming soon)_
36. `postgis` _(coming soon)_
37. `pgrouting` _(coming soon)_
38. `foreign-data-wrappers` _(coming soon)_
39. `postgrest` _(coming soon)_

**10. Server-Wide** (shares the whole server)

40. `logical-replication` _(coming soon)_
41. `incremental-backup` _(coming soon)_
42. `pg-cron` _(coming soon)_
43. `pgbouncer` _(coming soon)_

<!-- lessons:end -->

## Writing a lesson

Each lesson is one Markdown file, `lessons/<name>.md`, and `lessons/course.yaml` sets the order. See [`lessons/mvcc-bloat.md`](lessons/mvcc-bloat.md) for the format and [`docs/BUILD-SPEC.md`](docs/BUILD-SPEC.md) for the full design. Lessons are read from your checkout, so edits show up without a rebuild.

```bash
./tests/course.sh -v    # play every lesson and require its check to pass
./tests/phase1.sh       # drive the real interface through tmux
./scripts/readme        # regenerate the lesson list above
```

## Not included

- **TimescaleDB:** no PostgreSQL 19 package was available from the official Postgres repository when this was written.
- **pg_upgrade:** a real demo needs two full Postgres versions, which is too heavy for a one-minute lesson.

## Safety

This is a learning playground. It uses passwordless local auth and publishes no ports. Don't copy `compose.yaml` to production.

## License

MIT
