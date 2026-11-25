---
title: JSON_TABLE
tldr: Turn JSON into ordinary rows and columns with standard SQL, so you can filter, join, and insert it like any table.
---

```setup
CREATE TABLE orders_raw (id int PRIMARY KEY, payload jsonb);
INSERT INTO orders_raw VALUES
  (1, '{"customer": "Ada",   "items": [{"sku": "lamp", "qty": 2, "price": 30}, {"sku": "tent", "qty": 1, "price": 120}]}'),
  (2, '{"customer": "Linus", "items": [{"sku": "stove", "qty": 1, "price": 45}]}'),
  (3, '{"customer": "Grace", "items": [{"sku": "lamp", "qty": 1, "price": 30}, {"sku": "rope", "qty": 3, "price": 8}]}');
CREATE TABLE order_lines (customer text, sku text, qty int, price numeric);
```

```view
SELECT (SELECT count(*) FROM orders_raw) AS raw_orders,
       (SELECT count(*) FROM order_lines) AS order_lines,
       (SELECT coalesce(sum(qty * price), 0) FROM order_lines) AS revenue;
```

JSON often arrives as nested documents: an order holding a list of items, an API response holding a list of results. To report on it, you usually want plain rows, one per item, with ordinary columns.

**`JSON_TABLE`** does that. You give it a JSON value, a **path** that says which parts become rows, and a list of **columns**, each with its own path:

```sql
SELECT * FROM JSON_TABLE(
  '[{"player": "Ada", "points": 21}, {"player": "Linus", "points": 14}]'::jsonb,
  '$[*]' COLUMNS (player text PATH '$.player', points int PATH '$.points')
);
```

Paths use the SQL/JSON path language. `$` is the whole document, `.items` is a key, and `[*]` means every element of an array. So `'$[*]'` makes one row per array element, and `PATH '$.player'` reads that element's `player` key.

The result is a table, so everything SQL can do with a table works: `WHERE`, `JOIN`, `GROUP BY`, `INSERT INTO ... SELECT`.

**`NESTED PATH`** handles arrays inside documents. The outer path makes one row per document, and the nested path expands its array, repeating the outer columns on each row.

**Why not jsonb functions?** You can do the same with `jsonb_array_elements` and a `->>` and a cast for every field. It works, but it gets hard to read quickly, and it only works in Postgres. `JSON_TABLE` is standard SQL, so the same query works in Oracle, MySQL, and others.

**Where it came from.** `JSON_TABLE` is part of SQL/JSON, added to the SQL standard in 2016. Postgres got the path language in version 12 (2019). The rest of SQL/JSON was ready for version 15 but was pulled out late in that release cycle, and `JSON_TABLE` finally arrived in Postgres 17 (2024).

**The Live View** shows the raw orders and the table you'll fill in the challenge:

| Column | Meaning |
|---|---|
| `raw_orders` | Orders stored as JSON documents in `orders_raw` |
| `order_lines` | Rows in the plain `order_lines` table |
| `revenue` | `qty * price` summed over `order_lines` |

## Try it

Look at the raw data: each order is one document, with its items in an array.

```sql
SELECT id, payload FROM orders_raw WHERE id = 1;
```

```expect
"customer": "Ada"
```

Turn every order's items into rows. Listing `orders_raw o` first and then `JSON_TABLE(o.payload, ...)` runs the JSON_TABLE once for each order.

```sql
SELECT o.id, jt.*
FROM orders_raw o,
     JSON_TABLE(o.payload, '$.items[*]'
       COLUMNS (sku text PATH '$.sku', qty int PATH '$.qty', price numeric PATH '$.price')) jt;
```

```expect
stove |   1 |    45
(5 rows)
```

Add the customer to each row. The outer path `'$'` makes one row per order and reads `customer`. `NESTED PATH` then expands the items, repeating the customer on each.

```sql
SELECT jt.*
FROM orders_raw o,
     JSON_TABLE(o.payload, '$'
       COLUMNS (customer text PATH '$.customer',
                NESTED PATH '$.items[*]'
                  COLUMNS (sku text PATH '$.sku', qty int PATH '$.qty', price numeric PATH '$.price'))) jt;
```

```expect
Ada      | lamp
Grace    | rope  |   3 |     8
(5 rows)
```

## Challenge

Fill the plain table `order_lines` (customer, sku, qty, price) with one row per item from `orders_raw`. In the Live View, `order_lines` should reach 5 and `revenue` should reach 279.

Hint: Put INSERT INTO order_lines in front of the last step's query.

```solution
INSERT INTO order_lines
SELECT jt.*
FROM orders_raw o,
     JSON_TABLE(o.payload, '$'
       COLUMNS (customer text PATH '$.customer',
                NESTED PATH '$.items[*]'
                  COLUMNS (sku text PATH '$.sku', qty int PATH '$.qty', price numeric PATH '$.price'))) jt;
```

```check
SELECT count(*) = 5 AND sum(qty * price) = 279 AND count(*) FILTER (WHERE customer = 'Ada') = 2
FROM order_lines;
```
