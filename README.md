# postgres-demo

The best of modern Postgres in 27 tiny, runnable lessons. Each one takes about a minute: run it, read the output, move on.

Everything runs on **PostgreSQL 19** in a throwaway container. All you need is Docker.

## Start

```bash
./demo        # run the next lesson (repeat until done)
```

| Command | What it does |
|---|---|
| `./demo` | Run the next lesson |
| `./demo 17` | Run lesson 17 |
| `./demo list` | Show lessons and progress |
| `./demo all` | Run the whole course |
| `./demo psql` | Open a SQL prompt to experiment |
| `./demo down` | Remove the container and all its data |

Each lesson gets a fresh database, so you can rerun them in any order. To change one, edit its file in `lessons/` and run it again.

## Lessons

| # | Lesson | Type |
|---|---|---|
| 01 | [MVCC and bloat: why updates leave dead rows behind](lessons/01-mvcc-bloat.sql) | Built in |
| 02 | [Transactional DDL: undo a schema change](lessons/02-transactional-ddl.sql) | Built in |
| 03 | [SKIP LOCKED: a job queue with no extra software](lessons/03-skip-locked.sh) | Built in |
| 04 | [pg_stat_statements: find your slowest queries](lessons/04-pg-stat-statements.sql) | Built in |
| 05 | [JSONB: documents inside a relational database](lessons/05-jsonb.sql) | Built in |
| 06 | [JSON_TABLE (PG17): turn JSON into rows and columns](lessons/06-json-table.sql) | Built in |
| 07 | [Full-text search: built in since 8.3](lessons/07-full-text-search.sql) | Built in |
| 08 | [UUIDs: random v4 and time-ordered v7 (PG18)](lessons/08-uuid.sql) | Built in |
| 09 | [Generated columns: virtual by default (PG18)](lessons/09-generated-columns.sql) | Built in |
| 10 | [Index toolbox: partial, expression, BRIN, trigram, exclusion](lessons/10-index-toolbox.sql) | Built in |
| 11 | [Skip scan (PG18): use an index without its first column](lessons/11-skip-scan.sql) | Built in |
| 12 | [MERGE (PG15): insert, update, and delete in one statement](lessons/12-merge.sql) | Built in |
| 13 | [ON CONFLICT DO SELECT (PG19): get or create in one query](lessons/13-on-conflict-do-select.sql) | Built in |
| 14 | [Partitioning: split one big table into pieces](lessons/14-partitioning.sql) | Built in |
| 15 | [pg_partman: create future partitions automatically](lessons/15-pg-partman.sql) | Add-on |
| 16 | [Async I/O (PG18): read ahead instead of one page at a time](lessons/16-async-io.sql) | Built in |
| 17 | [REPACK (PG19): shrink a table while it stays online](lessons/17-repack.sh) | Built in |
| 18 | [pg_repack: the classic extension for older versions](lessons/18-pg-repack.sh) | Add-on |
| 19 | [pg_squeeze: the extension REPACK grew out of](lessons/19-pg-squeeze.sh) | Add-on |
| 20 | [Autovacuum insight (PG19): see what it will clean next](lessons/20-autovacuum.sql) | Built in |
| 21 | [Logical replication: copy live changes, now with sequences (PG19)](lessons/21-logical-replication.sh) | Built in |
| 22 | [Incremental backup (PG17): back up only what changed](lessons/22-incremental-backup.sh) | Built in |
| 23 | [pg_plan_advice (PG19): steer the query planner](lessons/23-plan-advice.sql) | Built in |
| 24 | [pgvector: similarity search for AI apps](lessons/24-pgvector.sql) | Add-on |
| 25 | [PostGIS: maps and distance in SQL](lessons/25-postgis.sql) | Add-on |
| 26 | [pg_cron: scheduled jobs inside the database](lessons/26-pg-cron.sh) | Add-on |
| 27 | [PgBouncer: many clients, few real connections](lessons/27-pgbouncer.sh) | Add-on |

"Built in" features ship with Postgres, including its bundled contrib modules. "Add-on" means a separate project, installed for you by the `Dockerfile`.

## Not included

- **TimescaleDB:** no PostgreSQL 19 package was available from the official Postgres repository when this was written.
- **pg_upgrade:** a real demo needs two full Postgres versions, which is too heavy for a one-minute lesson.

## Safety

This is a learning playground. It uses passwordless local auth, publishes no ports, and keeps data in memory. Don't copy `compose.yaml` to production.

## License

MIT
