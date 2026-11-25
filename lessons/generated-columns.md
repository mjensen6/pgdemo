---
title: Generated columns
tldr: A column Postgres computes from the rest of the row, so it can never be out of date. Virtual ones cost no disk space.
---

```setup
CREATE TABLE line_items (
  price numeric,
  qty   int,
  total numeric GENERATED ALWAYS AS (price * qty) VIRTUAL,
  taxed numeric GENERATED ALWAYS AS (price * qty * 1.08) STORED
);
INSERT INTO line_items VALUES (10, 3);
```

```view
SELECT price, qty, total, taxed FROM line_items;
```

A **generated column** is computed from other columns in the same row. You write the formula once, when you create the table, and Postgres keeps the value right from then on:

```sql
CREATE TABLE line_items (
  price numeric,
  qty   int,
  total numeric GENERATED ALWAYS AS (price * qty) VIRTUAL,
  taxed numeric GENERATED ALWAYS AS (price * qty * 1.08) STORED
);
```

You can't write to a generated column yourself. Change `price` or `qty`, and `total` and `taxed` follow.

**Why?** Without it, a total is kept in sync by app code or a trigger, and sooner or later some code path forgets. A generated column can't be forgotten.

**Two kinds, with different costs:**

- **`VIRTUAL`** computes the value every time a query reads it. It takes no disk space and makes writes no slower, but you can't index it. It's the default.
- **`STORED`** computes the value when the row is written and saves it on disk like a normal column. It takes space and makes writes a little slower, but reads are free and you can index it.

The formula can only use columns of the same row and functions that always give the same answer for the same input, so no `now()` and no other tables.

**Where it came from.** Generated columns are in the SQL standard. Postgres added `STORED` ones in version 12 (2019). `VIRTUAL` ones, and making them the default, came in Postgres 18 (2025). Before that, the usual tool was a trigger.

**The Live View** shows table `line_items`:

| Column | Meaning |
|---|---|
| `price`, `qty` | Normal columns you write |
| `total` | Virtual: `price * qty`, computed on read |
| `taxed` | Stored: `price * qty * 1.08`, saved on write |

## Try it

Change the quantity. In the Live View, `total` and `taxed` both follow at once, the virtual one because it's computed on read and the stored one because Postgres recomputed it on write.

```sql
UPDATE line_items SET qty = 5;
```

```expect
UPDATE 1
```

Try to set `total` by hand. Postgres refuses, because a generated column only ever holds its formula's result.

```sql
UPDATE line_items SET total = 100;
```

```expect
column "total" can only be updated to DEFAULT
```

Add a row. You only give `price` and `qty`. The generated columns fill themselves in.

```sql
INSERT INTO line_items (price, qty) VALUES (4, 2);
SELECT * FROM line_items;
```

```expect
50 | 54.00
8 |  8.64
```

## Challenge

Add a stored generated column called `with_shipping` to `line_items` that is always `price * qty + 5`.

Hint: ALTER TABLE line_items ADD COLUMN, then the same GENERATED ALWAYS AS (...) clause as `taxed`.

```solution
ALTER TABLE line_items ADD COLUMN with_shipping numeric GENERATED ALWAYS AS (price * qty + 5) STORED;
```

```check
SELECT EXISTS (SELECT FROM pg_attribute
               WHERE attrelid = 'line_items'::regclass AND attname = 'with_shipping' AND attgenerated = 's')
   -- to_jsonb reads the column by name without failing while it doesn't exist yet.
   AND (SELECT bool_and((to_jsonb(l) ->> 'with_shipping')::numeric = price * qty + 5) FROM line_items l);
```
