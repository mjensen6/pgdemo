# pgdemo

Learn modern PostgreSQL 19 in your terminal, one minute at a time. Each lesson runs a few real commands, shows you what changed, and ends with a small challenge you solve yourself.

All you need is Docker. Everything else, including Postgres, runs inside the course's own container and never touches anything already on your machine.

## Start

```bash
git clone https://github.com/mjensen6/pgdemo.git
cd pgdemo
./pgdemo
```

The first run checks Docker, downloads the course image (about 300 MB, once), and opens lesson 1. Run `./pgdemo` again any time to pick up where you left off.

The image is the one CI built and tested from the same code, for x86-64 and ARM, including Apple Silicon. If you change the Dockerfile or the player, `pgdemo` builds the image locally instead, which takes a few minutes. `PGDEMO_BUILD=1 ./pgdemo setup` always builds locally.

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
│ p prev lesson  n next lesson  ? help  q quit           │
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
pgdemo section 8        Start a Section (no number lists them)
pgdemo 5                Open lesson 5 (a name works too: pgdemo repack)
pgdemo 5 tldr           What it is and why it matters. Runs nothing
pgdemo 5 info           The full written lesson. Runs nothing
pgdemo 5 run            Run every step non-stop
pgdemo 5 view           Reopen the Live View
pgdemo 5 reset          Wipe the playground and return to step 1
pgdemo 5 check          Verify the lesson was completed
pgdemo 5 hint           Hint for the challenge
pgdemo 5 solution       Show the answer
pgdemo check            Re-verify every lesson you have opened
pgdemo browse           Explore the database with lazysql
pgdemo top              See live server activity with pg_activity
pgdemo psql             Open plain psql
pgdemo connect          Optional: connect a desktop app like TablePlus
pgdemo doctor           Diagnose problems in plain English
pgdemo stop             Stop the database
pgdemo uninstall        Remove the container, image, and saved data
pgdemo help             All commands, grouped by purpose
pgdemo help check       One command explained, with an example
```

Your progress is in the `progress` table of the `pgdemo` database, so you can query it with SQL.

## Lessons

<!-- lessons:start (generated from lessons/course.yaml) -->

**1. Core Concepts**

1. [MVCC and bloat](lessons/mvcc-bloat.md): Updates leave dead rows behind. VACUUM reuses the space but rarely shrinks the file.
2. [Transactional DDL](lessons/transactional-ddl.md): Schema changes can be rolled back like any other change, so a failed migration leaves nothing half done.

**2. Modern SQL**

3. [Window functions](lessons/window-functions.md): Rank, running totals, and comparisons with the previous row, computed across related rows without collapsing them the way GROUP BY does.
4. [Recursive queries](lessons/recursive-queries.md): WITH RECURSIVE walks trees and graphs, such as an org chart, a folder tree, or a comment thread, in a single query.
5. [MERGE](lessons/merge.md): Insert, update, and delete in one statement, by comparing a source table with a target table row by row.
6. [ON CONFLICT DO SELECT, new in Postgres 19](lessons/on-conflict-do-select.md): Get a row, or create it if it's missing, in one race-free statement that writes nothing when the row already exists.
7. [Generated columns](lessons/generated-columns.md): A column Postgres computes from the rest of the row, so it can never be out of date. Virtual ones cost no disk space.
8. [UUIDs, random and time-ordered](lessons/uuid.md): UUIDs are IDs you can make anywhere without asking the database. Time-ordered v7 ones keep indexes small and fast.
9. [Materialized views](lessons/materialized-views.md): Save a slow query's result as a table you can index, and refresh it when you choose, even without blocking readers.

**3. Documents and Search**

10. [JSONB](lessons/jsonb.md): Store JSON documents in a column and query inside them, with an index that makes lookups fast.
11. [JSON_TABLE](lessons/json-table.md): Turn JSON into ordinary rows and columns with standard SQL, so you can filter, join, and insert it like any table.
12. [Full-text search](lessons/full-text-search.md): Search text by meaning-carrying words, not exact characters, with ranking and a web-style query syntax. Built in, no search server needed.

**4. Indexes and Performance**

13. [The index toolbox](lessons/index-toolbox.md): Partial, expression, BRIN, and trigram indexes go beyond the plain B-tree, and exclusion constraints refuse overlapping bookings.
14. [Skip scan, new in Postgres 18](lessons/skip-scan.md): A multicolumn index now helps queries that leave out its first column, by hopping between that column's values.
15. [Asynchronous I/O, new in Postgres 18](lessons/async-io.md): Postgres now asks for many pages at once instead of waiting for each one, which speeds up big scans on slow disks.
16. [pg_stat_statements](lessons/pg-stat-statements.md): Postgres keeps a running tally of every query shape, so you can see which ones use the most time and fix those first.
17. [pg_plan_advice, new in Postgres 19](lessons/plan-advice.md): Write down a query plan as advice, then hand it back so the planner sticks to it. A safety valve for the rare plan that goes wrong.
18. [HypoPG](lessons/hypopg.md): Ask the planner whether an index would help before you spend the time and disk to build it.

**5. Queues and Events**

19. [SKIP LOCKED](lessons/skip-locked.md): Build a job queue on a plain table. Each worker grabs a job no one else holds, without waiting and without extra software.
20. [LISTEN and NOTIFY](lessons/listen-notify.md): One session sends a message and every session listening gets it at once, with no polling, and only when the sender's transaction commits.
21. [Advisory locks](lessons/advisory-locks.md): Lock on a number your app chooses, to make sure only one worker runs a job at a time, with no table to lock.
22. [River](lessons/river.md): A real Go job queue that keeps its jobs in a Postgres table, so enqueueing a job can be part of the same transaction as your data.

**6. Security and Testing**

23. [Row-level security](lessons/row-level-security.md): Two users query the same table and see different rows, because the database itself filters them, whatever query the app sends.
24. [pgaudit](lessons/pgaudit.md): A detailed log of who read or changed what, for the audits that security and compliance rules demand.
25. [pgTAP](lessons/pgtap.md): Unit tests written in SQL, for your functions, tables, and constraints, run inside the database and rolled back afterwards.

**7. Data at Scale**

26. [Partitioning](lessons/partitioning.md): Split one big table into pieces by date or value. Queries read only the pieces they need, and old data drops in an instant.
27. [pg_partman](lessons/pg-partman.md): An extension that creates future partitions before you need them and drops old ones on schedule, so a partitioned table runs itself.
28. [pg_ivm](lessons/pg-ivm.md): A materialized view that keeps itself up to date on every change, so it's fast to read and never stale.
29. [HyperLogLog](lessons/hll.md): Count unique visitors approximately, with about 2% error, in about a kilobyte, and combine days into weeks, which exact counts can't do.

**8. Keeping It Healthy**

30. [pg_activity](lessons/pg-activity.md): See what the server is doing right now, find the session that's blocking everyone else, and stop it.
31. [REPACK, new in Postgres 19](lessons/repack.md): REPACK rebuilds a bloated table to its real size. With CONCURRENTLY, the app keeps writing while it runs.
32. [pg_repack](lessons/pg-repack.md): The classic extension for rebuilding a bloated table while the app keeps running, on any Postgres version, not just 19.
33. [pg_squeeze](lessons/pg-squeeze.md): Rebuild a bloated table online from inside SQL, using logical decoding. It's the extension REPACK CONCURRENTLY grew out of.
34. [Autovacuum, and the scores new in Postgres 19](lessons/autovacuum.md): Autovacuum cleans dead rows for you once a table passes a threshold. Postgres 19 shows each table's score and cleans the neediest first.

**9. Beyond Tables**

35. [pgvector](lessons/pgvector.md): Store embeddings next to your data and find the most similar items with one ORDER BY. The core of search and recommendations in AI apps.
36. [PostGIS](lessons/postgis.md): Store places as real geography and ask spatial questions in SQL, such as how far apart two places are and what's within 300 miles.
37. [pgRouting](lessons/pgrouting.md): Find the shortest route through a road network in SQL, and see it change when a road closes.
38. [Foreign data wrappers](lessons/foreign-data-wrappers.md): Query a table in another database as if it were local, join it with your own tables, and let Postgres send the filtering to the other side.
39. [PostgREST](lessons/postgrest.md): Turn tables into a REST API with no code. Postgres permissions decide what the API allows.

**10. Server-Wide** (shares the whole server)

40. [Logical replication, now with sequences](lessons/logical-replication.md): Stream row changes from one database to another as they happen. Postgres 19 can copy sequences too, so a switchover doesn't reuse IDs.
41. [Incremental backup](lessons/incremental-backup.md): Back up only the blocks that changed since the last backup, then combine the pieces into one you can restore. Built into Postgres 17.
42. [pg_cron](lessons/pg-cron.md): Schedule SQL to run by itself, every few seconds or every night at 3, inside the database, with no cron server to manage.
43. [PgBouncer](lessons/pgbouncer.md): A connection pooler that lets thousands of app connections share a handful of real Postgres connections.

<!-- lessons:end -->

## Recipes

[`recipes/`](recipes/README.md) holds ready-to-run SQL for everyday jobs, outside the lessons: diffing two tables or two databases, finding duplicates, table sizes and bloat, unused and missing indexes, who's blocking whom, and replication lag. Each is a plain `.sql` file you can run on any database. To try one in pgdemo, open `pgdemo psql`, then:

```
\i /opt/pgdemo/recipes/sample-data.sql
\i /opt/pgdemo/recipes/data/diff-two-tables.sql
```

## Writing a lesson

Each lesson is one Markdown file, `lessons/<name>.md`, and `lessons/course.yaml` sets the order. See [`lessons/mvcc-bloat.md`](lessons/mvcc-bloat.md) for the format and [`docs/BUILD-SPEC.md`](docs/BUILD-SPEC.md) for the full design. Lessons are read from your checkout, so edits show up without a rebuild.

```bash
./tests/course.sh -v    # play every lesson: pins, Live View, and check must all pass
./tests/phase1.sh       # drive the real interface through tmux, from a clean start
./tests/phase2.sh       # test the tools through tmux (keeps your progress)
./scripts/readme        # regenerate the lesson list above
```

## Not included

- **TimescaleDB:** no PostgreSQL 19 package was available from the official Postgres repository when this was written.
- **pg_upgrade:** a real demo needs two full Postgres versions, which is too heavy for a one-minute lesson.

## Safety

This is a learning playground. It uses passwordless local auth and publishes no ports. `pgdemo connect` opens one, on 127.0.0.1 only, until `pgdemo connect off`. Don't copy `compose.yaml` to production.

## License

MIT
