# Backlog

Ideas for lessons and extensions that aren't in the course yet. None of these is planned or promised. Before building one, check that it works on Postgres 19 and that any extension has a `postgresql-19-*` package, or builds cleanly in the Dockerfile.

To start a lesson, add its name to `lessons/course.yaml`. It shows as "coming soon" until `lessons/<name>.md` exists.

## Top picks

These five would turn the course from a feature tour into something that prepares people to run Postgres in production.

1. **Functions and triggers.** No lesson teaches them, yet `pgtap` tests functions and `pg-ivm` is built on triggers.
2. **Locks and safe migrations.** Covers `lock_timeout`, the queue that builds up behind `ALTER TABLE`, and adding a constraint `NOT VALID` then running `VALIDATE`. It links `transactional-ddl` to `pg-activity`.
3. **Roles and privileges.** `row-level-security` and `postgrest` both depend on `GRANT` without teaching it.
4. **Reading EXPLAIN.** A dozen lessons use it, and none explains how to read it. This could be lesson 3.
5. **Point-in-time recovery.** `incremental-backup` makes the backups but never restores to a moment in time. For example, undo a `DROP TABLE` from five minutes ago.

## New section: Code in the Database

- **SQL and PL/pgSQL functions**, including `SECURITY DEFINER` and its risks.
- **Triggers**, such as an audit-trail table and keeping `updated_at` current.
- **Procedures that COMMIT partway through** (11), for batched backfills.

## More lessons for existing sections

| Section | Lesson | Notes |
|---|---|---|
| Core Concepts | Isolation levels and `SERIALIZABLE` (9.1) | Show a write-skew bug, then show `SERIALIZABLE` catching it. |
| Core Concepts | Reading EXPLAIN | See the top picks. |
| Modern SQL | `LATERAL`, `FILTER`, `DISTINCT ON`, `GROUPING SETS` | One "SQL you didn't know" lesson. |
| Modern SQL | `RETURNING OLD` / `NEW` (18) | Pairs with `merge`. |
| Modern SQL | Range types (9.2) and multiranges (14) | Comes before temporal keys. |
| Modern SQL | Temporal keys with `WITHOUT OVERLAPS` (18) | The standard-SQL follow-up to the exclusion constraint in `index-toolbox`. |
| Indexes and Performance | Extended statistics, `CREATE STATISTICS` (10) | Fixes bad estimates on correlated columns. Goes next to `plan-advice`. |
| Indexes and Performance | Covering indexes with `INCLUDE` (11), and `CREATE INDEX CONCURRENTLY` | Could fold into `index-toolbox`. |
| Security and Testing | Roles and privileges | Default privileges, and predefined roles such as `pg_read_all_data`. |
| Security and Testing | pgcrypto | Hash passwords and encrypt a column. It's in contrib. |
| Keeping It Healthy | Locks and safe migrations | See the top picks. |
| Keeping It Healthy | `pg_stat_io` (16) and wait events | Ties to `async-io`. |
| Keeping It Healthy | Transaction ID wraparound | Why autovacuum can never be turned off. |
| Server-Wide | Streaming replication and failover | A hot standby, read queries on it, and `pg_promote()`. |
| Server-Wide | Point-in-time recovery | See the top picks. |
| Server-Wide | Change data capture with logical decoding | Uses `test_decoding` or wal2json. Explains how `pg-squeeze` and `REPACK CONCURRENTLY` work underneath. |
| Server-Wide | Major-version upgrades with pg_upgrade | Probably too heavy for a one-minute lesson. |

Also search for other Postgres 19 features worth a lesson. The course covers REPACK, `ON CONFLICT DO SELECT`, plan advice, autovacuum scores, sequence replication, and the I/O worker settings.

## Extensions to consider

Each one needs adding to the image, `courseExtensions` in `player/tools.go`, and the doctor check. Check each one's license and how much it grows the image.

| Extension | What it does | Fits in |
|---|---|---|
| **pgmq** | A message queue with visibility timeouts, all in SQL. | Queues and Events, next to `river`. |
| **pg_search** (ParadeDB) | Full-text search with BM25 ranking, as in Elasticsearch. Check its license: it may be AGPL. | Documents and Search, after `full-text-search`. |
| **pg_hint_plan** | Steers the planner with hints in comments. It's the long-standing extension that plan advice now partly replaces. | Indexes and Performance, next to `plan-advice`. |
| **pg_stat_monitor** (Percona) | `pg_stat_statements` plus time buckets, histograms, and query plans. | Indexes and Performance, after `pg-stat-statements`. |
| **pg_graphql** (Supabase) | A GraphQL API made from your schema, queried from SQL. | Beyond Tables, next to `postgrest`. |
| **pg_net** (Supabase) | Makes HTTP requests from SQL without blocking, for example to call a webhook from a trigger. | Beyond Tables, or Queues and Events. |
| **Apache AGE** | Graph queries in openCypher, on top of Postgres tables. | Beyond Tables, after `recursive-queries` as the graph alternative. |

Considered and set aside:

- **pg_duckdb**: fast analytics. Worth another look if its image size and Postgres 19 support work out.
- **Citus**: too heavy for a one-container course.
- **TimescaleDB**: its best features aren't under an open-source license.
