---
title: ON CONFLICT DO SELECT, new in Postgres 19
tldr: Get a row, or create it if it's missing, in one race-free statement that writes nothing when the row already exists.
---

```setup
CREATE TABLE tags (id serial PRIMARY KEY, name text UNIQUE);
INSERT INTO tags (name) VALUES ('postgres'), ('sql');
```

```view
SELECT id, name, xmin AS written_by_tx FROM tags ORDER BY id;
```

A very common need is **get or create**: give me the row for tag `postgres`, and create it first if it doesn't exist. Apps do this for tags, users signing in for the first time, and lookup tables of every kind.

**The naive way has a race.** `SELECT` first, then `INSERT` if nothing came back. If two sessions do this at the same moment, both see nothing, both insert, and one fails on the unique constraint.

**`INSERT ... ON CONFLICT`** fixes the race. The insert either succeeds or hits the unique constraint, and you say what to do on a conflict. Before version 19 there were two choices, and neither fit:

- `DO NOTHING` skips the insert, but `RETURNING` then returns no row, so you need a second query to get the `id`.
- `DO UPDATE SET name = EXCLUDED.name` returns the row, but it really updates it. That writes a new row version every time, leaving a dead row behind (the MVCC lesson) and firing any update triggers, just to read an `id`.

**`DO SELECT`** returns the existing row and writes nothing. It needs a `RETURNING` clause, since returning the row is its whole point. Add `FOR UPDATE` to also lock the row you get back.

**Where it came from.** `INSERT ... ON CONFLICT`, often called upsert, arrived in Postgres 9.5 (2016) with `DO NOTHING` and `DO UPDATE`. `DO SELECT` is new in Postgres 19 (2026). People had asked for it ever since 9.5, because the `DO UPDATE` workaround was so common.

**The Live View** shows table `tags`:

| Column | Meaning |
|---|---|
| `id` | The tag's ID, from a sequence |
| `name` | The tag, which must be unique |
| `written_by_tx` | The transaction that wrote this row version. It changes whenever the row is rewritten |

## Try it

`DO NOTHING` first. The tag `postgres` already exists, so nothing is inserted, and nothing comes back either. You'd need a second query to find its `id`.

```sql
INSERT INTO tags (name) VALUES ('postgres') ON CONFLICT (name) DO NOTHING RETURNING id, name;
```

The `DO UPDATE` workaround does return the row. But in the Live View, `written_by_tx` for `postgres` changes: Postgres wrote a whole new row version just to answer a question.

```sql
INSERT INTO tags (name) VALUES ('postgres') ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id, name;
```

`DO SELECT` returns the existing row and writes nothing, so `written_by_tx` stays the same this time.

```sql
INSERT INTO tags (name) VALUES ('postgres') ON CONFLICT (name) DO SELECT RETURNING id, name;
```

For a tag that doesn't exist yet, the same statement creates it and returns the new row. Notice its `id` is 6, not 3: each of the three attempts above took a number from the sequence before it found the conflict. Gaps in IDs are normal and harmless.

```sql
INSERT INTO tags (name) VALUES ('mvcc') ON CONFLICT (name) DO SELECT RETURNING id, name;
```

## Challenge

Get or create the tags `sql` and `indexing` in one statement, returning both of their ids. `sql` already exists, so it must not be rewritten.

Hint: One INSERT can take several rows: VALUES ('a'), ('b'). End it like the last step.

```solution
INSERT INTO tags (name) VALUES ('sql'), ('indexing') ON CONFLICT (name) DO SELECT RETURNING id, name;
```

```check
-- indexing was created, and sql was not rewritten: its row version is still
-- older than the one the DO UPDATE step wrote for postgres.
SELECT EXISTS (SELECT FROM tags WHERE name = 'indexing')
   AND (SELECT xmin::text::bigint FROM tags WHERE name = 'sql')
     < (SELECT xmin::text::bigint FROM tags WHERE name = 'postgres');
```
