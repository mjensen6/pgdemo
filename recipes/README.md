# Recipes

Ready-to-run SQL for everyday Postgres jobs: diffing data, finding what's slow or bloated, and seeing who's blocking whom. Each file does one job. A comment at the top says what it does, when to use it, what to change, and whether it writes anything.

They're plain SQL, so they work on any database, not just pgdemo. They're tested on Postgres 19, and most work on much older versions. A recipe that needs a newer column says which version.

## Run one

On your own database:

```sh
psql -d mydb -f recipes/health/table-sizes.sql
```

Or copy the query into any SQL client. Most recipes use example table names, such as `customers_old`. Change them to your own, as the comment at the top says.

## Try them on sample data

`sample-data.sql` creates a `recipes_demo` database with a small table for every recipe to find something in. It also creates `recipes_remote`, for the cross-database diff, and a `report_reader` role. Running it again starts over.

pgdemo can see this folder at `/opt/pgdemo/recipes`. Open `pgdemo psql`, then:

```
\i /opt/pgdemo/recipes/sample-data.sql
\i /opt/pgdemo/recipes/data/diff-two-tables.sql
```

The sample data leaves you connected to `recipes_demo`, so every recipe after it runs there. Next time, `\c recipes_demo` first. To remove it all: `DROP DATABASE recipes_demo; DROP DATABASE recipes_remote; DROP ROLE report_reader;`

## Data

| Recipe | What it does |
|---|---|
| [diff-two-tables](data/diff-two-tables.sql) | Rows added, removed, or changed between two tables with the same key |
| [diff-which-columns](data/diff-which-columns.sql) | For each changed row, only the columns that differ, old and new |
| [diff-except](data/diff-except.sql) | Rows in one table and not the other, with no key needed |
| [diff-across-databases](data/diff-across-databases.sql) | Diff a table against its copy in another database, through postgres_fdw |
| [checksum-blocks](data/checksum-blocks.sql) | Find which blocks of rows differ between two big tables, without diffing every row |
| [find-duplicates](data/find-duplicates.sql) | Values that appear more than once, and the ids of their rows |
| [remove-duplicates](data/remove-duplicates.sql) | Delete duplicates, keeping the oldest. **Writes**, after a dry run |
| [exact-row-counts](data/exact-row-counts.sql) | Exact row counts for every table, next to the planner's estimate |
| [gaps-and-islands](data/gaps-and-islands.sql) | Runs of consecutive days, such as login streaks |
| [keyset-pagination](data/keyset-pagination.sql) | Page through a big table quickly, without OFFSET |
| [batched-update](data/batched-update.sql) | Update millions of rows in batches, committing as it goes. **Writes** |
| [batched-delete](data/batched-delete.sql) | Delete old rows in batches, committing as it goes. **Writes** |

## Health

| Recipe | What it does |
|---|---|
| [table-sizes](health/table-sizes.sql) | The biggest tables, split into data, indexes, and TOAST |
| [bloat-estimate](health/bloat-estimate.sql) | How much of each table is dead rows and free space |
| [unused-indexes](health/unused-indexes.sql) | Indexes no query has used, which only cost writes and disk |
| [duplicate-indexes](health/duplicate-indexes.sql) | Indexes that are exact copies of another |
| [fk-without-index](health/fk-without-index.sql) | Foreign keys with no index, and the CREATE INDEX to fix each |
| [cache-hit-ratio](health/cache-hit-ratio.sql) | How often reads come from memory, per database and table |
| [xid-age](health/xid-age.sql) | How close each database and table is to transaction ID wraparound |
| [sequences-running-out](health/sequences-running-out.sql) | Serial and identity columns closest to their maximum |
| [slowest-queries](health/slowest-queries.sql) | The queries that use the most total time, from pg_stat_statements |

## Activity

| Recipe | What it does |
|---|---|
| [long-running-queries](activity/long-running-queries.sql) | Queries running longer than a minute |
| [idle-in-transaction](activity/idle-in-transaction.sql) | Sessions holding a transaction open while doing nothing |
| [blocking-locks](activity/blocking-locks.sql) | Who is waiting on a lock, and who is blocking them |
| [stop-session](activity/stop-session.sql) | The command to cancel a query or end a session, for each one running |

## Schema

| Recipe | What it does |
|---|---|
| [mostly-null-columns](schema/mostly-null-columns.sql) | Columns that are almost always NULL |
| [tables-without-primary-key](schema/tables-without-primary-key.sql) | Tables with no primary key |
| [list-constraints](schema/list-constraints.sql) | Every constraint, written out as SQL |
| [table-privileges](schema/table-privileges.sql) | Who can do what on each table |

## Replication

| Recipe | What it does |
|---|---|
| [replication-lag](replication/replication-lag.sql) | How far behind each replica and subscriber is |
| [slots-holding-wal](replication/slots-holding-wal.sql) | Replication slots, and how much WAL each keeps on disk |

## Adding a recipe

Put it in the folder that fits, with the comment header the others have, and add it to the table above. Then add what its output should contain to `tests/recipes.sh`, and run that test. It fails if a recipe is missing from this list or from the test.
