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

Start a transaction, drop a column, and create a table. The Live View is another session, so it still shows the old schema: nothing is visible to others until you commit.

```sql
BEGIN;
ALTER TABLE users DROP COLUMN name;
CREATE TABLE oops (id int);
SELECT * FROM users;
```

Changed your mind? In Postgres, ROLLBACK undoes schema changes too. The column and its data come back, and table `oops` never existed.

```sql
ROLLBACK;
SELECT * FROM users;
SELECT to_regclass('oops') AS oops_table;
```

## Challenge

In one transaction, add a column `email` of type `text` to `users` and create a table called `audit` with a column `id int`. Then make it stick.

Hint: Start with BEGIN and finish with COMMIT. The check runs in another session, so it only sees committed changes.

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
