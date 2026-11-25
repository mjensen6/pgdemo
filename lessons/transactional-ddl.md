---
title: Transactional DDL
tldr: Schema changes can be rolled back like any other change, so a failed migration leaves nothing half done.
---

```setup
CREATE TABLE users (id int, name text);
INSERT INTO users VALUES (1, 'Ada');
```

```view
SELECT table_name AS "table", string_agg(column_name, ', ' ORDER BY ordinal_position) AS columns
FROM information_schema.columns
WHERE table_schema = 'public'
GROUP BY table_name ORDER BY 1;
```

**DDL**, short for data definition language, is the SQL that changes the shape of the database rather than the data in it: `CREATE TABLE`, `ALTER TABLE`, `DROP TABLE`, `CREATE INDEX` and so on.

A **transaction** groups several statements into one all-or-nothing change. `BEGIN` starts it. `COMMIT` makes every change stick at once. `ROLLBACK` throws every change away, as if none of it happened. Until you commit, no other session can see what you did.

In Postgres, DDL is **transactional**: it follows those same rules. You can create tables, drop columns and add indexes inside a transaction, then commit them together or roll them all back.

**Why does this matter?** A schema migration is usually several steps. If step three fails, you don't want steps one and two left behind, with a half-migrated database that the app can't use and nobody can easily fix. With transactional DDL, a failed migration rolls back cleanly, and the app never sees an in-between state.

**How it works.** Postgres stores its list of tables and columns, called the **system catalog**, in ordinary tables. Changing the schema means changing rows in those tables, so it gets the same MVCC treatment you saw in the last lesson: new row versions that only become visible on commit.

**The limits.** A few commands can't run inside a transaction, because they do work that can't be undone or take a long time on purpose. `VACUUM`, `CREATE DATABASE` and the `CONCURRENTLY` versions of commands are the common ones. Postgres refuses them with a clear error instead of half running them. Also, `ALTER TABLE` locks the table until you commit or roll back, so keep transactions that change busy tables short.

**Where it came from.** Postgres has worked this way since its early days, inherited from the Berkeley POSTGRES project it grew out of. It wasn't added in a later release or adopted from an extension. Not every database does this. Oracle and MySQL commit automatically before and after each DDL statement, so a failed migration there can leave a half-changed schema behind.

**The Live View** runs in its own session, so it only shows committed changes:

| Column | Meaning |
|---|---|
| `table` | Each table in the playground |
| `columns` | That table's columns, in order |

## Try it

Start a transaction, drop a column, and create a table. The query at the end shows that inside your transaction the `name` column is already gone. The Live View, in another session, still shows the old schema: `users` keeps its `name` column and there's no `oops` table.

```sql
BEGIN;
ALTER TABLE users DROP COLUMN name;
CREATE TABLE oops (id int);
SELECT * FROM users;
```

```expect
ALTER TABLE
CREATE TABLE
(1 row)
```

Changed your mind? `ROLLBACK` undoes schema changes too. The column and its data come back, and table `oops` never existed. `to_regclass` looks a table up by name and returns nothing when it doesn't exist.

```sql
ROLLBACK;
SELECT * FROM users;
SELECT to_regclass('oops') AS oops_table;
```

```expect
1 | Ada
```

Some commands refuse to run inside a transaction. `VACUUM` is one of them, so Postgres stops it with an error and changes nothing. After an error the transaction is aborted, so `ROLLBACK` ends it.

```sql
BEGIN;
VACUUM users;
ROLLBACK;
```

```expect
VACUUM cannot run inside a transaction block
ROLLBACK
```

## Challenge

In one transaction, add a column `email` of type `text` to `users` and create a table called `audit` with a column `id int`. Then make it stick.

Hint: Start with BEGIN, like the first step, but finish with COMMIT instead of ROLLBACK. The check runs in another session, so it only sees committed changes.

```solution
BEGIN;
ALTER TABLE users ADD COLUMN email text;
CREATE TABLE audit (id int);
COMMIT;
```

```check
SELECT to_regclass('audit') IS NOT NULL
   AND EXISTS (SELECT FROM information_schema.columns
               WHERE table_name = 'users' AND column_name = 'email');
```
