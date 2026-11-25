---
title: UUIDs, random and time-ordered
tldr: UUIDs are IDs you can make anywhere without asking the database. Time-ordered v7 ones keep indexes small and fast.
---

```setup
CREATE TABLE ids_v4 (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), n int);
CREATE TABLE ids_v7 (id uuid PRIMARY KEY DEFAULT uuidv7(), n int);
```

```view
SELECT 'v4 random' AS kind, count(*) AS rows, pg_size_pretty(pg_relation_size('ids_v4_pkey')) AS index_size FROM ids_v4
UNION ALL
SELECT 'v7 time-ordered', count(*), pg_size_pretty(pg_relation_size('ids_v7_pkey')) FROM ids_v7;
```

A **UUID** is a 128-bit ID, written like `0195a3c4-7e2b-7d61-9f3a-5c8e1b2d4f60`. There are so many possible values that any program can make one without asking the database, and it will never clash with anyone else's. That's handy when IDs are created in the app, on phones that go offline, or across several databases.

There are two kinds worth knowing:

- **Version 4** is completely random. `gen_random_uuid()` makes one.
- **Version 7** starts with the current time in milliseconds, then adds random bits. `uuidv7()` makes one. IDs made later sort after IDs made earlier.

**Why the order matters.** A primary key has a B-tree index, which keeps its entries sorted. Random v4 IDs land all over the index, so every insert touches a different page. Pages split and end up half empty, the index grows larger, and less of it fits in memory. Time-ordered v7 IDs all land at the end of the index, the way a plain counter does, so pages fill up neatly.

**The cost of v7** is that anyone who sees the ID can read when it was made. If creation time is private, use v4.

You can set either one as a column's default:

```sql
CREATE TABLE ids_v7 (id uuid PRIMARY KEY DEFAULT uuidv7(), n int);
```

**Where it came from.** The `uuid` type is old, but making UUIDs used to need an extension, `uuid-ossp` or `pgcrypto`. `gen_random_uuid()` moved into core Postgres in version 13 (2020). Version 7 was defined by RFC 9562 in 2024, and `uuidv7()` arrived in Postgres 18 (2025).

**The Live View** compares two tables that are the same except for the kind of ID:

| Column | Meaning |
|---|---|
| `kind` | Which kind of UUID the table's `id` uses |
| `rows` | Rows in the table |
| `index_size` | Size of the primary key index |

## Try it

Make three of each. The v4 IDs share nothing. The v7 IDs made in the same instant start with the same characters, because they start with the time.

```sql
SELECT gen_random_uuid() AS v4, uuidv7() AS v7 FROM generate_series(1, 3);
```

```expect
(3 rows)
```

Because a v7 ID carries its creation time, Postgres can read it back out.

```sql
SELECT uuid_extract_timestamp(uuidv7()) AS created_at, now() AS now;
```

```expect
created_at
```

Insert 200,000 rows into each table. Watch `index_size` in the Live View: the random v4 index ends up noticeably bigger for the same number of rows.

```sql
INSERT INTO ids_v4 (n) SELECT generate_series(1, 200000);
INSERT INTO ids_v7 (n) SELECT generate_series(1, 200000);
```

```expect
INSERT 0 200000
INSERT 0 200000
```

## Challenge

Create a table called `events` with a `uuid` primary key named `id` that gets a time-ordered UUID by default, and a `note text` column.

Hint: Copy the CREATE TABLE example from the intro, with the new table and column names.

```solution
CREATE TABLE events (id uuid PRIMARY KEY DEFAULT uuidv7(), note text);
```

```check
SELECT coalesce((SELECT pg_get_expr(d.adbin, d.adrelid) = 'uuidv7()'
                 FROM pg_attrdef d JOIN pg_attribute a ON a.attrelid = d.adrelid AND a.attnum = d.adnum
                 WHERE d.adrelid = to_regclass('events') AND a.attname = 'id'), false)
   AND EXISTS (SELECT FROM pg_index WHERE indrelid = to_regclass('events') AND indisprimary);
```
