---
title: MERGE
tldr: Insert, update, and delete in one statement, by comparing a source table with a target table row by row.
---

```setup
CREATE TABLE stock (sku text PRIMARY KEY, qty int);
INSERT INTO stock VALUES ('apple', 5), ('pear', 2);
CREATE TABLE delivery (sku text, qty int);
INSERT INTO delivery VALUES ('apple', 10), ('pear', -2), ('plum', 7);
CREATE TABLE delivery2 (sku text, qty int);
INSERT INTO delivery2 VALUES ('apple', -15), ('plum', 3), ('kiwi', 4);
```

```view
SELECT sku, qty FROM stock ORDER BY sku;
```

A common job is syncing one table into another: a nightly import, a batch of changes from another system, a delivery that changes stock levels. Some incoming rows match existing rows and should update them. Some are new and should be inserted. Some mean an existing row should go.

**MERGE** does all of that in one statement. You name a **target** (the table to change) and a **source** (the incoming rows), say how they match, then list what to do in each case:

```sql
MERGE INTO stock s USING delivery d ON s.sku = d.sku
WHEN MATCHED AND s.qty + d.qty <= 0 THEN DELETE
WHEN MATCHED THEN UPDATE SET qty = s.qty + d.qty
WHEN NOT MATCHED THEN INSERT VALUES (d.sku, d.qty);
```

Postgres checks the `WHEN` clauses in order for each source row and runs the first one that fits. `RETURNING merge_action()` tells you which action each row got.

**Why one statement?** Doing it as separate `UPDATE`, `INSERT`, and `DELETE` statements means three passes over the data and three places to get the matching rule wrong. One `MERGE` reads the source once and applies one rule.

**The limits.** MERGE isn't a safe way for many sessions to insert the same key at once. Two sessions can both decide a row is "not matched" and both try to insert it, and one fails with a unique violation. For a single-row insert-or-update under heavy concurrency, use `INSERT ... ON CONFLICT`, from the next lesson.

**Where it came from.** MERGE is part of the SQL standard since SQL:2003, and Oracle and SQL Server had it for years. Postgres added it in version 15 (2022). Before that, people used `INSERT ... ON CONFLICT` (9.5, 2016) or chains of statements. Version 17 (2024) added `RETURNING` and `merge_action()`.

**The Live View** shows table `stock`, the target:

| Column | Meaning |
|---|---|
| `sku` | The product |
| `qty` | How many are in stock |

## Try it

Here's the source: a delivery that adds 10 apples, takes away 2 pears, and brings 7 plums, a product `stock` doesn't have yet.

```sql
SELECT * FROM delivery;
```

```expect
(3 rows)
```

Merge the delivery into stock. Apples get updated to 15. Pears drop to 0, so the first rule deletes them. Plums don't match anything, so they're inserted. Watch all three happen at once in the Live View. `RETURNING` shows which action each row got, with the row as it was before a delete.

```sql
MERGE INTO stock s USING delivery d ON s.sku = d.sku
WHEN MATCHED AND s.qty + d.qty <= 0 THEN DELETE
WHEN MATCHED THEN UPDATE SET qty = s.qty + d.qty
WHEN NOT MATCHED THEN INSERT VALUES (d.sku, d.qty)
RETURNING merge_action(), s.*;
```

```expect
UPDATE       | apple |  15
DELETE       | pear  |   2
INSERT       | plum  |   7
```

## Challenge

A second delivery arrived in table `delivery2`: it takes away 15 apples, adds 3 plums, and brings 4 kiwis. Merge it into `stock` with the same rules: add the amounts, insert new products, and delete any product that drops to zero or below.

Hint: Use the same MERGE as the last step, with `delivery2` as the source.

```solution
MERGE INTO stock s USING delivery2 d ON s.sku = d.sku
WHEN MATCHED AND s.qty + d.qty <= 0 THEN DELETE
WHEN MATCHED THEN UPDATE SET qty = s.qty + d.qty
WHEN NOT MATCHED THEN INSERT VALUES (d.sku, d.qty);
```

```check
-- After both deliveries: apples gone, plums 7 + 3, kiwis new.
SELECT coalesce(array_agg(sku || '=' || qty ORDER BY sku) = '{kiwi=4,plum=10}', false) FROM stock;
```
