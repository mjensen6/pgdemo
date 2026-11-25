---
title: The index toolbox
tldr: Partial, expression, BRIN, and trigram indexes go beyond the plain B-tree, and exclusion constraints refuse overlapping bookings.
---

```setup
CREATE EXTENSION pg_trgm;
CREATE EXTENSION btree_gist;
CREATE TABLE users AS
  SELECT i AS id, 'User' || i || '@Mail.com' AS email, i % 100 = 0 AS active,
         now() - i * interval '1 minute' AS created
  FROM generate_series(1, 100000) i;
ANALYZE users;
CREATE TABLE bookings (room int, during tstzrange);
```

```view
SELECT c.relname AS index, am.amname AS kind, pg_size_pretty(pg_relation_size(c.oid)) AS size
FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid JOIN pg_am am ON am.oid = c.relam
WHERE i.indrelid IN ('users'::regclass, 'bookings'::regclass)
ORDER BY pg_relation_size(c.oid);
```

An index is a separate structure that lets Postgres find rows without reading the whole table. `CREATE INDEX` makes a **B-tree** by default, which is right for most columns: it's sorted, so it answers `=`, `<`, `>`, and `ORDER BY`. But a few other tools solve problems a plain B-tree can't, or solve them far more cheaply:

- **Partial index**: add `WHERE` to index only some rows. If queries only ever look for active users, index just those, and the index is a tiny fraction of the size.
- **Expression index**: index a computed value, such as `lower(email)`. A query that searches `lower(email) = '...'` can use it. A plain index on `email` can't help that query.
- **BRIN**, short for block range index: for data whose order on disk follows a column, like timestamps in an append-only log. It stores only the lowest and highest value per range of pages, so it's tiny.
- **Trigram index**, from the `pg_trgm` extension: splits text into three-letter pieces so `LIKE '%middle%'` can use an index. A B-tree only helps when the pattern starts with fixed text.
- **Exclusion constraint**: a rule enforced by an index, generalizing `UNIQUE`. "No two bookings for the same room may overlap" is one constraint, and the database refuses the second booking.

**The cost of every index**: it takes disk space, and every insert and update has to maintain it. Add the indexes your queries need, not every one you can think of.

**Where it came from.** The B-tree and GiST index frameworks come from the Berkeley POSTGRES project. Partial and expression indexes have been in Postgres since the early 2000s, and GIN, which trigram indexes use, since 8.2 (2006). Exclusion constraints arrived in 9.0 (2010), and BRIN in 9.5 (2016). `pg_trgm` is a contrib extension that ships with Postgres. Version 9.1 (2011) let it speed up `LIKE` and `ILIKE`.

**The Live View** lists the indexes on `users` and `bookings`, smallest first:

| Column | Meaning |
|---|---|
| `index` | The index name |
| `kind` | Its type: `btree`, `brin`, `gin`, or `gist` |
| `size` | Disk space it takes. `users` has 100,000 rows |

## Try it

A partial index on active users only. 1 in 100 users is active, so watch how small it is in the Live View.

```sql
CREATE INDEX users_active ON users (id) WHERE active;
```

An expression index on `lower(email)`, so a case-insensitive lookup can use an index. The plan finds the rows through `users_lower`.

```sql
CREATE INDEX users_lower ON users (lower(email));
EXPLAIN (COSTS OFF) SELECT id FROM users WHERE lower(email) = 'user5@mail.com';
```

```expect
users_lower
```

A BRIN index on `created`, which goes steadily back in time as the table goes on. It covers all 100,000 rows in a few kilobytes.

```sql
CREATE INDEX users_brin ON users USING brin (created);
```

A trigram index makes a search for text in the middle of a value use an index. The plan uses `users_trgm` instead of reading the whole table.

```sql
CREATE INDEX users_trgm ON users USING gin (email gin_trgm_ops);
EXPLAIN (COSTS OFF) SELECT id FROM users WHERE email LIKE '%er4242@%';
```

```expect
Bitmap Index Scan on users_trgm
```

Now an exclusion constraint: no two bookings of the same room may overlap in time. `&&` means "overlaps". The first booking goes in. The second overlaps it, so Postgres refuses it with an error. That error is the point.

```sql
ALTER TABLE bookings ADD EXCLUDE USING gist (room WITH =, during WITH &&);
INSERT INTO bookings VALUES (1, '[2030-01-01 09:00, 2030-01-01 10:00)');
INSERT INTO bookings VALUES (1, '[2030-01-01 09:30, 2030-01-01 11:00)');
```

```expect
ALTER TABLE
INSERT 0 1
conflicting key value violates exclusion constraint
```

## Challenge

The app's login only looks up active users, by `lower(email)`. Create one index that serves exactly that query and nothing more, so it's as small as possible.

Hint: Combine two of the tools: the expression from users_lower and the WHERE from users_active.

```solution
CREATE INDEX users_active_lower ON users (lower(email)) WHERE active;
```

```check
-- An index on users that has both an expression and a WHERE clause.
SELECT EXISTS (SELECT FROM pg_index
               WHERE indrelid = 'users'::regclass
                 AND pg_get_expr(indexprs, indrelid) LIKE '%lower(email)%'
                 AND pg_get_expr(indpred, indrelid) = 'active');
```
