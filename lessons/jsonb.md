---
title: JSONB
tldr: Store JSON documents in a column and query inside them, with an index that makes lookups fast.
---

```setup
CREATE TABLE products (id serial PRIMARY KEY, doc jsonb);
INSERT INTO products (doc) VALUES
  ('{"name": "Lamp", "tags": ["home"], "price": 30}'),
  ('{"name": "Tent", "tags": ["outdoor", "camping"], "price": 120}'),
  ('{"name": "Stove", "tags": ["outdoor", "camping"], "price": 45, "fuel": "gas"}');
CREATE INDEX ON products USING gin (doc);
```

```view
SELECT id, doc->>'name' AS name, doc->>'price' AS price, doc - 'name' - 'price' AS other_keys
FROM products ORDER BY id;
```

**`jsonb`** is a column type that holds a whole JSON document. Each row can have different keys, so it suits data whose shape varies: product attributes, settings, event payloads, or a response from another API.

Postgres parses the JSON when you write it and stores it in a binary form. That makes reading inside it fast, but it doesn't keep your exact formatting, key order, or duplicate keys. The older **`json`** type keeps the text exactly as written and parses it again on every read. Use `jsonb` unless you need the original text back.

A few operators do most of the work:

| Operator | Meaning | Example |
|---|---|---|
| `->` | Get a key, as JSON | `doc->'tags'` gives `["home"]` |
| `->>` | Get a key, as text | `doc->>'name'` gives `Lamp` |
| `@>` | Contains | `doc @> '{"tags": ["camping"]}'` |

To change one key, `jsonb_set(doc, '{price}', '99')` returns the document with that key replaced. It adds the key if it's missing.

**A GIN index** on the column makes `@>` fast on big tables. It indexes every key and value in every document, so one index covers any containment query.

**The costs.** Nothing checks the shape of your documents unless you add constraints, so a typo like `"prcie"` goes in silently. And changing one key rewrites the whole document as a new row version, the MVCC rule from lesson 1, so huge documents that change often are expensive. Columns you always query are usually better as normal columns, with `jsonb` for the rest.

**Where it came from.** It began as an extension: `hstore`, a contrib module for flat key-value pairs, since Postgres 8.2 (2006). The `json` type arrived in 9.2 (2012). `jsonb` followed in 9.4 (2014), built by the same people behind `hstore` and their work on a nested version of it. It made Postgres a serious choice for document data.

**The Live View** shows table `products`:

| Column | Meaning |
|---|---|
| `name`, `price` | Pulled out of each document with `->>` |
| `other_keys` | The rest of the document. `-` removes a key |

## Try it

Pull fields out of the documents. `->>` returns text, so cast it to compare as a number.

```sql
SELECT doc->>'name' AS name, (doc->>'price')::int AS price FROM products WHERE (doc->>'price')::int < 100;
```

```expect
Lamp
Stove
(2 rows)
```

Find every product tagged `camping`. `@>` asks whether the document contains this piece of JSON, and the GIN index answers it.

```sql
SELECT doc->>'name' AS name FROM products WHERE doc @> '{"tags": ["camping"]}';
```

```expect
Tent
Stove
(2 rows)
```

Change one key in place. Watch the Tent's `price` change in the Live View while its other keys stay.

```sql
UPDATE products SET doc = jsonb_set(doc, '{price}', '99') WHERE doc->>'name' = 'Tent';
```

```expect
UPDATE 1
```

## Challenge

The Stove is on sale. Add a key `sale` with the value `true` to the Stove's document, and leave the other products alone.

Hint: jsonb_set adds a key that isn't there yet. The JSON value true is written '{sale}', 'true'.

```solution
UPDATE products SET doc = jsonb_set(doc, '{sale}', 'true') WHERE doc->>'name' = 'Stove';
```

```check
SELECT (SELECT doc @> '{"sale": true, "fuel": "gas"}' FROM products WHERE doc->>'name' = 'Stove')
   AND NOT EXISTS (SELECT FROM products WHERE doc ? 'sale' AND doc->>'name' <> 'Stove');
```
