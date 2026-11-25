---
title: pgvector
tldr: Store embeddings next to your data and find the most similar items with one ORDER BY. The core of search and recommendations in AI apps.
---

```setup
CREATE EXTENSION vector;
CREATE TABLE docs (id serial PRIMARY KEY, title text, embedding vector(3));
INSERT INTO docs (title, embedding) VALUES
  ('Dog training tips',   '[0.9, 0.1, 0.0]'),
  ('Best pizza dough',    '[0.0, 0.9, 0.1]'),
  ('Postgres indexing',   '[0.0, 0.0, 1.0]'),
  ('Robot pet review',    '[0.6, 0.0, 0.7]'),
  ('Cat food guide',      '[0.6, 0.7, 0.0]'),
  ('Baking with Python',  '[0.0, 0.6, 0.7]');
```

```view
SELECT d.title, d.embedding, round((d.embedding <=> '[1, 0, 0.1]')::numeric, 3) AS distance_to_query
FROM docs d ORDER BY d.embedding <=> '[1, 0, 0.1]';
```

AI apps often need **similarity search**: find the documents closest in meaning to a question, the products most like this one, the support answers that fit a ticket.

They do it with **embeddings**. An AI model turns a piece of text, or an image, into a **vector**: a list of numbers, usually hundreds or thousands of them, where similar meanings get similar numbers. Finding similar items then means finding the nearest vectors.

**pgvector** adds a `vector` column type to Postgres, along with operators that measure distance:

| Operator | Distance | Index operator class |
|---|---|---|
| `<=>` | Cosine: compares direction, ignores length. The usual choice for text embeddings | `vector_cosine_ops` |
| `<->` | Euclidean: straight-line distance | `vector_l2_ops` |
| `<#>` | Negative inner product, for embeddings already normalized to length 1 | `vector_ip_ops` |

Smaller always means more similar. A search is just `ORDER BY embedding <=> query LIMIT 5`.

**Indexes for speed.** Without an index, Postgres compares the query with every row. An **HNSW** index builds a graph that links each vector to its near neighbors, so a search hops through the graph instead. An index only helps a query that uses **its** distance, so pick the operator class that matches your operator.

**The trade-off.** HNSW is **approximate**: very fast, but now and then it misses a true nearest neighbor. That's almost always fine for search. Vectors are also big, since 1,536 numbers take about 6 kB per row.

**Where it came from.** pgvector is a third-party extension, started by Andrew Kane in 2021. HNSW indexes came in version 0.5.0 (2023). It isn't part of core Postgres, but nearly every hosted Postgres offers it. It made "keep your vectors next to your data" a real alternative to a separate vector database.

The vectors here have just 3 numbers, so you can read them: roughly [about animals, about food, about tech].

**The Live View** ranks every document by cosine distance to the query `[1, 0, 0.1]`, which means "mostly animals". Closest first.

## Try it

Find the 3 documents closest to "mostly animals". The dog training tips come first, then the robot pet and the cat food guide.

```sql
SELECT title, round((embedding <=> '[1, 0, 0.1]')::numeric, 3) AS distance
FROM docs ORDER BY embedding <=> '[1, 0, 0.1]' LIMIT 3;
```

Search for "food and tech" instead. Different query, different winners.

```sql
SELECT title, round((embedding <=> '[0, 0.7, 0.7]')::numeric, 3) AS distance
FROM docs ORDER BY embedding <=> '[0, 0.7, 0.7]' LIMIT 3;
```

Add an HNSW index for cosine distance. With 6 rows Postgres would rather read the table, so turn sequential scans off for a moment to see the plan use the index.

```sql
CREATE INDEX docs_cosine ON docs USING hnsw (embedding vector_cosine_ops);
SET enable_seqscan = off;
EXPLAIN (COSTS OFF) SELECT title FROM docs ORDER BY embedding <=> '[1, 0, 0.1]' LIMIT 3;
RESET enable_seqscan;
```

## Challenge

A new feature will search with Euclidean distance, `<->`, which the cosine index can't help. Create the HNSW index those searches need.

Hint: Same as the last step's CREATE INDEX, with the operator class for <-> from the intro's table.

```solution
CREATE INDEX docs_l2 ON docs USING hnsw (embedding vector_l2_ops);
```

```check
SELECT EXISTS (SELECT FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid
               JOIN pg_am am ON am.oid = c.relam JOIN pg_opclass oc ON oc.oid = i.indclass[0]
               WHERE i.indrelid = 'docs'::regclass AND am.amname = 'hnsw' AND oc.opcname = 'vector_l2_ops');
```
