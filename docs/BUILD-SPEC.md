# pgdemo build spec

This is the complete brief for building `pgdemo`. It assumes you have not seen the design conversation. Use the terms in [`GLOSSARY.md`](../GLOSSARY.md) exactly.

## Goal

`pgdemo` is an open-source, terminal-only course that teaches modern PostgreSQL 19 through 43 one-minute lessons. A developer who knows basic `SELECT` but little about Postgres should go from `git clone` to a working lesson with one command, and without reading setup docs.

The bar for every decision is: **would this be obvious to someone who did not build it?** Keep lessons short. Do not drag on about any feature.

## Current state

- **All five phases are done.** `./pgdemo`, the Go player, the three-pane screen, progress, and every command in [Commands](#commands) work.
- **All 43 lessons are built**: the 27 from the original course, converted, and the 16 new ones. The player still supports "coming soon" for any lesson added to `course.yaml` before its file exists.
- The original course's scripts were the source for Phase 3, and have been removed. They're still in `main`'s history.
- `tests/course.sh`, `tests/phase1.sh`, and `tests/phase2.sh` pass on PostgreSQL 19.0.
- The repo is public at https://github.com/mjensen6/pgdemo. The course was built on branch `design/approachable-cli` and merged into `main` with the owner's approval.
- The original 27-lesson course is the first commit on `main`, `Add 27-lesson PostgreSQL 19 feature course`.
- The old container `postgres-demo-db-1` was removed with the owner's approval.
- Ideas for more lessons and extensions are in [`BACKLOG.md`](BACKLOG.md).

## Build order

Stop at the end of each phase and hand it to the owner. Do not begin the next phase until they give feedback.

1. **Slice.** _Done._ `./pgdemo`, the three-pane layout, the player, progress, and these 3 lessons: `mvcc-bloat`, `transactional-ddl`, `repack`. All 43 lessons appear in `pgdemo list`, and unbuilt ones show as "coming soon".
2. **Tools.** _Done._ All remaining commands, help, `doctor`, `browse`, `top`, and `connect`.
3. **Convert** _Done._ The remaining 24 existing lessons to the new format.
4. **Add** _Done._ The 16 new lessons.
5. **Publish** to GitHub only after the owner's final approval. _Done._ The repo is public, CI is live and publishes the prebuilt image, commits are authored by the owner's GitHub no-reply address, and the folder is named `pgdemo`.

## Architecture

```
host                                 container (postgres:19 + tools)
./pgdemo  (thin Bash launcher) ───▶  pgdemo-player (Go)
  - Docker checks                      - lesson parsing and playback
  - compose up/stop                    - progress, playgrounds, checks
  - PATH prompt                        - tmux layout
                                       - Postgres, pgcli, pspg, lazysql, pg_activity
```

- **`./pgdemo`** is a small Bash script at the repo root. It only deals with the host and Docker, then forwards every other command to the player. It replaces `./demo`.
- **The player** is written in Go and compiled during the image build. Learners never need Go.
- **Always use the course's own private Postgres.** Never detect, connect to, or modify a Postgres already on the machine. Do not ask whether Postgres is installed.
- **tmux runs inside the container**, so learners do not need to install it. It must work when the learner is already inside their own tmux, because the owner works that way.
- Install and run third-party tools only inside the Docker image, never on the host.

## First run

Plain `pgdemo` runs `setup` automatically when needed, so a newcomer never has to know that command exists.

`setup` must:

1. Check Docker is installed. If not, show the right install command for the OS and offer to run it after confirmation.
2. Check the Docker daemon is running and the user can reach it.
3. Handle a stale Docker group. If the account is in the `docker` group but the current shell is not, run Docker through `sg docker -c ...` and explain in one plain sentence. This really happened: long-lived desktop and tmux sessions keep their old groups even after being added.
4. Pull the prebuilt image for this exact code, falling back to a local build if there isn't one. `PGDEMO_BUILD=1` always builds locally.
5. Ask once whether to add `pgdemo` to PATH, by linking it into `~/.local/bin`. Never change PATH silently.
6. Open the first lesson on its **Course Overview** page, which explains the Lesson pane, SQL pane, Live View, and tabs inside the real screen. Its text lives in `lessons/overview.md`.

## Screen layout

Assume a full-screen, wide terminal. Do not build medium or phone layouts. If the terminal is under about 120x30, show a one-line suggestion to enlarge it and continue anyway.

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

- **Three panes:** Lesson, SQL, and Live View. The Live View is on by default.
- **Tabs:** Browse and Top are full-screen tmux windows, shown in a clickable tab bar.
- **Mouse on.** Clicking switches panes and tabs. Nobody should need to learn the tmux prefix.
- **Lesson pane:** renders Markdown and highlighted code with Glamour. A footer always shows the keys.
- **SQL pane:** pgcli, already connected to the current playground, so nobody types `\c`. Fall back to `psql` if pgcli does not work with Postgres 19.
- **Live View pane:** pspg in watch mode, running the lesson's `view` query. Lessons with no view show a short placeholder.
- Changing lessons reconnects the SQL and Live View panes automatically.

### Keys in the Lesson pane

| Key | Action |
|---|---|
| Enter | Run the step on screen, then move to the next one. On the challenge, check it |
| b | Back one step |
| r | Reset the lesson, after confirmation: rebuild the playground, reconnect the panes, and go back to the start |
| c | Check |
| h | Hint |
| s | Skip the challenge |
| p | Previous lesson |
| n | Next lesson |
| ? | Full key and command help |
| q | Quit, after confirmation |

`q` asks for confirmation, stops the database, and prints how to resume: `Run pgdemo to pick up where you left off.`

## Commands

```
pgdemo                  Start, or resume where you left off
pgdemo list             All lessons by Section, with progress
pgdemo section 8        Start a Section (no number lists them)
pgdemo 5                Open lesson 5 in the guided panes
pgdemo repack           Same, by lesson name
pgdemo 5 tldr           What it is and why it matters, in 3 lines. Runs nothing
pgdemo 5 info           The full written lesson. Runs nothing
pgdemo 5 run            Run every step non-stop
pgdemo 5 view           Reopen the Live View
pgdemo 5 reset          Wipe the playground and return to step 1
pgdemo 5 check          Verify the lesson was completed
pgdemo 5 hint           Hint for the challenge
pgdemo 5 solution       Show the answer
pgdemo check            Re-verify every lesson you have opened
pgdemo browse           Explore the database with lazysql (pgdemo browse 2 picks a lesson)
pgdemo top              See live server activity with pg_activity
pgdemo psql             Open plain psql
pgdemo connect          Optional: connect a desktop app like TablePlus (pgdemo connect off undoes it)
pgdemo setup            Check Docker and prepare everything
pgdemo doctor           Diagnose problems in plain English
pgdemo stop             Stop the database
pgdemo uninstall        Remove the container, image, and saved data
pgdemo help             All commands, grouped by purpose
pgdemo help check       One command explained, with an example
```

- `setup` is course-wide only. `reset` is the only per-lesson way to rebuild a playground.
- Every lesson command accepts a number or a name.
- `uninstall` must confirm first, since it deletes progress.

### `pgdemo connect`

This is fully optional and sets up nothing until first used. The course never downloads TablePlus or any desktop app.

On first use, it writes `compose.override.yaml`, which publishes `127.0.0.1:5439` to Postgres, and recreates the container, keeping the saved volume. Docker Compose merges that file automatically, so the port stays open across restarts until `pgdemo connect off` deletes it. The file is in `.gitignore`. It prints a ready-to-paste string such as `postgresql://postgres@localhost:5439/<playground>`, for the lesson named or the one worked on last.

The owner chose ease of use here, so **no password is generated**. Keep the port bound to `127.0.0.1` so it is never reachable from other machines.

## Lessons

### Identity and order

`lessons/course.yaml` defines the Sections and their lesson order. Each lesson is `lessons/<name>.md`.

- **The name is the permanent ID**, and progress is stored by name.
- **Numbers are computed** from the order in `course.yaml`, running 1–43 across Sections, so adding a lesson never breaks saved progress.

### Format

Each lesson is one Markdown file that also reads well on GitHub.

````markdown
---
title: MVCC and bloat
tldr: Updates leave dead rows behind. VACUUM reuses the space but rarely shrinks the file.
---

```setup
CREATE EXTENSION pgstattuple;
CREATE TABLE t (id int, note text) WITH (autovacuum_enabled = false);
INSERT INTO t SELECT i, 'hello' FROM generate_series(1, 10000) i;
```

```view
SELECT pg_size_pretty(pg_table_size('t')) AS size, tuple_count AS rows,
       dead_tuple_count AS dead FROM pgstattuple('t');
```

An UPDATE writes a new row and leaves the old one dead.

```sql
UPDATE t SET note = 'world';
```

## Challenge

Set every note to 'done' without making `t` any bigger.

Hint: What gave you free space before?

```solution
VACUUM t;
UPDATE t SET note = 'done';
```

```check
SELECT pg_table_size('t') < 1000000 AND NOT EXISTS (SELECT FROM t WHERE note <> 'done');
```
````

| Block | Purpose |
|---|---|
| `setup` | Hidden. Prepares the playground |
| `view` | The Live View query |
| Text before `## Try it` | The Intro: the teaching, including what the Live View shows. Code blocks here are examples and never run |
| Paragraph + `sql` or `sh` | One step, after `## Try it` |
| `## Challenge` | The task the learner does themselves |
| `Hint:` line | Challenge hint |
| `solution` | The answer |
| `check` | Hidden query returning one boolean |
| `expect` | Optional, right after a step's command. Hidden text its output must contain, line by line, in order. The course test checks it |
| `cleanup` | Optional. Undoes server-wide effects |

Every lesson needs a `tldr`, a challenge, a hint, a solution, and a check.

A challenge practices what the lesson just taught. It may combine the steps in a new way, but the solution must use only commands and ideas the lesson explained. Never make the learner discover a new command from the hint.

The Intro follows the MVCC lesson's shape: what the feature is, why it works that way, what it costs or where it stops, a **Where it came from** paragraph, and a table explaining each Live View column. **Where it came from** gives the Postgres version and year the feature arrived, and says whether it started outside core, as a contrib module or a third-party extension that became common enough to be built in. Check these facts against release notes or commit messages, not memory.

### Intro and steps

A lesson teaches first and practices second. The **Intro** explains the idea and what the Live View will show, and runs nothing. The steps after `## Try it` are the hands-on part.

Nothing runs without the learner seeing it first. A step opens with its explanation and the exact command under "Press Enter to run this SQL". Enter runs it, and the pane then shows "You ran" and the result. The next Enter moves on. Each step's text should say what the command does and what to watch for in the Live View.

The player shows each step's output in the Lesson pane. The SQL pane stays the learner's own.

All `sql` steps in a lesson must share **one persistent database session**. Lessons rely on this, for example `BEGIN` in one step and `ROLLBACK` in a later one. Use one long-running `psql` process per lesson so output matches what learners see in real Postgres.

### Pins

When a step's text makes a claim about its output, such as which row comes first or that Postgres refuses with an error, pin it with an `expect` block. Pin only the words the claim depends on, never timings, random IDs, or process IDs, so a pin breaks only when the lesson stops telling the truth. The course test fails a lesson whose output no longer matches its pins.

### Checks

A check verifies the **result in the playground, never the typed text**, so any correct approach passes. A failed check says "not yet" and offers the hint. `s` skips the challenge.

### Playgrounds

- Each lesson has its own database, named `lesson_<name>` with hyphens replaced by underscores.
- A playground is created the first time its lesson opens. Do not build them all in advance.
- It keeps the learner's work until `reset`, including across restarts.
- `reset` rewinds to step 1 and rebuilds the playground. It does not remove a lesson's `done` status.

### Server-wide lessons

Replication, scheduled jobs, and server settings belong to the whole server, so they cannot be fully isolated. These lessons are grouped in Section 10 and labeled as sharing the server, both in `pgdemo list` and in the lesson itself. They must run their `cleanup` block when the learner leaves or resets. `pgdemo doctor` flags leftovers such as cron jobs, replication slots, and subscriptions.

### Course order

Sources marked with a file name were scripts in the original course, on `main`.

| # | Section | Lesson name | Source |
|---|---|---|---|
| 1 | 1 Core Concepts | `mvcc-bloat` | `01-mvcc-bloat.sql` |
| 2 | 1 Core Concepts | `transactional-ddl` | `02-transactional-ddl.sql` |
| 3 | 2 Modern SQL | `window-functions` | new |
| 4 | 2 Modern SQL | `recursive-queries` | new |
| 5 | 2 Modern SQL | `merge` | `12-merge.sql` |
| 6 | 2 Modern SQL | `on-conflict-do-select` | `13-on-conflict-do-select.sql` |
| 7 | 2 Modern SQL | `generated-columns` | `09-generated-columns.sql` |
| 8 | 2 Modern SQL | `uuid` | `08-uuid.sql` |
| 9 | 2 Modern SQL | `materialized-views` | new |
| 10 | 3 Documents and Search | `jsonb` | `05-jsonb.sql` |
| 11 | 3 Documents and Search | `json-table` | `06-json-table.sql` |
| 12 | 3 Documents and Search | `full-text-search` | `07-full-text-search.sql` |
| 13 | 4 Indexes and Performance | `index-toolbox` | `10-index-toolbox.sql` |
| 14 | 4 Indexes and Performance | `skip-scan` | `11-skip-scan.sql` |
| 15 | 4 Indexes and Performance | `async-io` | `16-async-io.sql` |
| 16 | 4 Indexes and Performance | `pg-stat-statements` | `04-pg-stat-statements.sql` |
| 17 | 4 Indexes and Performance | `plan-advice` | `23-plan-advice.sql` |
| 18 | 4 Indexes and Performance | `hypopg` | new |
| 19 | 5 Queues and Events | `skip-locked` | `03-skip-locked.sh` |
| 20 | 5 Queues and Events | `listen-notify` | new |
| 21 | 5 Queues and Events | `advisory-locks` | new |
| 22 | 5 Queues and Events | `river` | new |
| 23 | 6 Security and Testing | `row-level-security` | new |
| 24 | 6 Security and Testing | `pgaudit` | new |
| 25 | 6 Security and Testing | `pgtap` | new |
| 26 | 7 Data at Scale | `partitioning` | `14-partitioning.sql` |
| 27 | 7 Data at Scale | `pg-partman` | `15-pg-partman.sql` |
| 28 | 7 Data at Scale | `pg-ivm` | new |
| 29 | 7 Data at Scale | `hll` | new |
| 30 | 8 Keeping It Healthy | `pg-activity` | new |
| 31 | 8 Keeping It Healthy | `repack` | `17-repack.sh` |
| 32 | 8 Keeping It Healthy | `pg-repack` | `18-pg-repack.sh` |
| 33 | 8 Keeping It Healthy | `pg-squeeze` | `19-pg-squeeze.sh` |
| 34 | 8 Keeping It Healthy | `autovacuum` | `20-autovacuum.sql` |
| 35 | 9 Beyond Tables | `pgvector` | `24-pgvector.sql` |
| 36 | 9 Beyond Tables | `postgis` | `25-postgis.sql` |
| 37 | 9 Beyond Tables | `pgrouting` | new |
| 38 | 9 Beyond Tables | `foreign-data-wrappers` | new |
| 39 | 9 Beyond Tables | `postgrest` | new |
| 40 | 10 Server-Wide | `logical-replication` | `21-logical-replication.sh` |
| 41 | 10 Server-Wide | `incremental-backup` | `22-incremental-backup.sh` |
| 42 | 10 Server-Wide | `pg-cron` | `26-pg-cron.sh` |
| 43 | 10 Server-Wide | `pgbouncer` | `27-pgbouncer.sh` |

What the new lessons show:

| Lesson | The one-minute point |
|---|---|
| `window-functions` | Rank and running totals without collapsing rows |
| `recursive-queries` | Walk a tree, like an org chart |
| `materialized-views` | Save a slow query's result and refresh it |
| `hypopg` | Test whether an index would help without building it |
| `listen-notify` | One session sends a message and another receives it |
| `advisory-locks` | Coordinate app workers with a lock on a number |
| `river` | A real Go job queue on Postgres |
| `row-level-security` | Two users query one table and see different rows |
| `pgaudit` | A log of who read or changed what |
| `pgtap` | Unit tests written in SQL |
| `pg-ivm` | A materialized view that updates itself |
| `hll` | Count unique visitors approximately in tiny storage |
| `pg-activity` | Spot a running query and what is blocking it |
| `pgrouting` | Shortest route between places |
| `foreign-data-wrappers` | Query another database as if it were local |
| `postgrest` | Turn a table into a REST API with no code |

## Progress

Store progress in a dedicated database named `pgdemo`, in a table named `progress`, so learners can query it with SQL.

```sql
CREATE TABLE progress (
  lesson       text PRIMARY KEY,
  state        text NOT NULL CHECK (state IN ('watched', 'done')),
  current_step int  NOT NULL DEFAULT 1,
  updated_at   timestamptz NOT NULL DEFAULT now()
);
```

A lesson with no row is **not started**.

| Symbol | State | Meaning |
|---|---|---|
| `○` | Not started | Never opened |
| `◐` | Watched | Steps ran, including through `run`, but no check has passed |
| `●` | Done | A check passed |

Only a passed check marks a lesson done.

## Docker

- Base image: `postgres:19`, using its Debian packages.
- **The prebuilt image.** CI tests on x86-64 and ARM runners. On a push to `main`, each runner publishes the exact image it tested to `ghcr.io/mjensen6/pgdemo:src-<hash>-<arch>`, and a final job joins them into `ghcr.io/mjensen6/pgdemo:src-<hash>` and `:latest`. `<hash>` comes from `scripts/source-hash`, over everything built into the image: the Dockerfile, the player, and the River demo. The launcher computes the same hash, so a checkout pulls only the image built from its own code, and builds locally when there's none. Lessons aren't in the hash, since they're read from the checkout at run time.
- Add: tmux, pgcli, pspg, pg-activity, and the player. Build lazysql in a Go build stage at a pinned version (v0.7.0). Build it from source, so no telemetry endpoint is compiled in, and also pass `--no-telemetry` and set `DO_NOT_TRACK=1` and `LAZYSQL_NO_UPDATE_CHECK=1`, so it never contacts the network.
- Add extensions: `postgresql-19-` plus `cron`, `partman`, `pgvector`, `postgis-3`, `postgis-3-scripts`, `repack`, `squeeze`, `hypopg`, `pg-ivm`, `pgrouting`, `hll`, `pgaudit`, `pgtap`, and the `pgbouncer` package.
- PostgREST is not in those packages. The image downloads a pinned release (v13.0.8) and checks its SHA-256. Only x86-64 has a static build. ARM uses the Ubuntu build, which is untested.
- River needs a small Go demo program, `river-demo/`, compiled into the image in its own build stage, since River needs Go 1.26. It retries failed jobs after 2 seconds, so the lesson doesn't wait.
- curl is installed for the PostgREST lesson.
- Set the compose project name to `pgdemo`.
- Keep the whole server on a named Docker volume so playgrounds and progress survive restarts. This replaces the current in-memory `tmpfs`.
- Publish no ports by default. Only `pgdemo connect` opens one.
- Authentication stays local `trust`. This is a learning playground.

Required server settings:

```
shared_preload_libraries=pg_stat_statements,pg_cron,pg_squeeze,pgaudit
cron.database_name=postgres
cron.use_background_workers=on
max_worker_processes=32
summarize_wal=on
output_plugin_libraries=pgoutput,test_decoding,pg_squeeze
```

## Known problems already solved

- **pg_squeeze:** Postgres 19 blocks it with `may not be used as an output plugin` unless it is in `output_plugin_libraries`.
- **pg_cron:** jobs silently fail with "connection failed" without `cron.use_background_workers=on`. A lesson can look fine while running zero jobs, so assert the job actually ran.
- **VACUUM** cannot run in a multi-statement `psql -c`. Run it as its own statement.
- **No error does not mean a lesson works.** Read the real output of each lesson. Lesson 26's old version passed with zero job runs.
- **Exclusion constraint error** in `index-toolbox` is intentional.
- **`REPACK (CONCURRENTLY)`** requires a primary key and cannot run inside a transaction block.
- **Stray processes crash the server.** Postgres treats an unknown child that dies by signal as a crash and restarts. Without an init process, any orphan in the container is handed to the postmaster. `compose.yaml` sets `init: true`, and `sh` steps that start a background job must `kill` it and `wait` for it.
- **A background job in an `sh` step can hang the step** if it keeps the step's output open, as `{ ...; sleep 600; } | psql &` does. Redirect the whole background pipeline to `/dev/null`. The player also stops waiting one second after the step's own shell exits.
- **Some lessons need roles**, which belong to the whole server: `rls_alice`, `rls_bob`, `pgaudit_auditor`, and `web_anon`. Their setup creates them only if they don't exist, and they're named after their lessons.
- **Postgres 19 replaced `io_workers`** with `io_min_workers` and `io_max_workers`.
- **A check must fail on a fresh playground.** Lesson 1's original size check passed before any step ran. The course test now requires every check to fail on a fresh playground and before the solution, and to pass after it.

## Verify these before relying on them

These come from project docs or memory and have not been run here:

- pgcli works with Postgres 19, and Enter runs a statement by default.
- pspg's documented `--query` and `--watch` work for the Live View with Postgres 19.
- lazysql and pg_activity work with Postgres 19.
- The new extension packages install and load together on the `postgres:19` image.
- The nested tmux case handles mouse and keys when the learner is already in tmux.

If one fails, choose the simplest substitute that keeps the experience the same, and tell the owner.

Phase 1 result: all of the above were verified on `postgres:19` (19.0, Debian trixie) pgcli 4.3.0, pspg 5.9.1, pg_activity 3.6.2, and all 14 extensions load together. Nested tmux passes mouse clicks for panes and tabs.

Phase 2 result: lazysql v0.7.0 works with Postgres 19 over the container's local TCP connection, in the Browse tab and through `pgdemo browse`. It asks "Exit LazySQL?" before quitting.

## Testing

- Test on Linux. The owner will try macOS. The README must say Linux is tested and ask for reports from others.
- Drive the real interface in tests with tmux `send-keys` and `capture-pane`. Do not only unit test the parser.
- `tests/phase1.sh` starts from a clean state in a Compose project of its own, `pgdemo-test`, with its own volume and state folder (`PGDEMO_STATE`), so it never touches the tester's course or progress. `tests/phase2.sh` runs against the tester's own course and keeps its progress.
- A course test runs every lesson's steps and solution from a clean start, then requires its check to pass.
- Add a simple GitHub Actions workflow that runs the course test. Keep it simple for now.
- Generate the README's numbered lesson list from `course.yaml`, and fail the tests if it is out of date.

### Phase 1 is done when

From a clean state with no `pgdemo` container or volume:

1. `./pgdemo` builds, starts, and opens lesson 1 in three panes with no manual steps.
2. Enter steps through the lesson, and the Live View numbers change as steps run.
3. The SQL pane is already connected, and the learner can type the challenge there.
4. `c` fails before the challenge is solved and passes after, and `pgdemo list` then shows `●`.
5. `pgdemo 2` keeps one session across steps, so its `ROLLBACK` works.
6. `pgdemo repack` shows the rebuild with its background writer still landing writes.
7. `pgdemo 1 tldr`, `info`, `reset`, `hint`, and `solution` all work.
8. `q` confirms, stops the database, and prints how to resume. Running `pgdemo` again restores progress and the playground.
9. All of the above also works when launched from inside an existing tmux session.

Then stop, tell the owner how to try it, and wait for feedback.

### Phase 2 is done when

`tests/phase2.sh` checks each of these:

1. `pgdemo help` lists every command above, and `pgdemo help check` explains one with an example. No command says it "arrives in the next build".
2. `pgdemo section` lists the Sections, `pgdemo section 99` says which Sections exist, and `pgdemo section 8` opens one of its lessons.
3. The Browse tab runs lazysql on the playground and follows the lesson when the learner presses `n` or `p`.
4. `pgdemo 1 view`, `pgdemo top`, and `pgdemo browse 2` each open full screen and quit cleanly.
5. `pgdemo connect 2` opens `127.0.0.1:5439` only, a real client can query through it, and `pgdemo connect off` closes it.
6. `pgdemo doctor` passes on a healthy setup and explains a leftover replication slot with the command to remove it.
7. `pgdemo uninstall` asks first, and answering no removes nothing.

## Repo and publishing rules

- Project, command, and image name: `pgdemo`. License: MIT.
- Work on a branch and open a pull request. Ask the owner before pushing to `main` or publishing anything new, such as the prebuilt image.
- The owner's personal email must never appear in commits. Commits use the owner's GitHub no-reply address, `mjensen6 <64800027+mjensen6@users.noreply.github.com>`, set in this repo's git config.
- Keep `GLOSSARY.md` up to date when a term changes.
