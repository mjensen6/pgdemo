---
title: Full-text search
tldr: Search text by meaning-carrying words, not exact characters, with ranking and a web-style query syntax. Built in, no search server needed.
---

```setup
CREATE TABLE articles (
  id     serial PRIMARY KEY,
  body   text,
  food   bool DEFAULT false,
  search tsvector GENERATED ALWAYS AS (to_tsvector('english', body)) STORED
);
INSERT INTO articles (body) VALUES
  ('Running a database on a home server'),
  ('The runner finished the marathon'),
  ('How to bake sourdough bread'),
  ('Baking with kids on a rainy day'),
  ('The breadth of features in Postgres'),
  ('Why the database ran out of disk');
CREATE INDEX ON articles USING gin (search);
```

```view
SELECT id, food, body, search FROM articles ORDER BY id;
```

`LIKE '%run%'` finds the characters "run" anywhere, so it also matches "brunch", it misses "ran", it can't tell you which result fits best, and it can't use a normal index. Real search needs something smarter.

**Full-text search** works on words instead of characters. `to_tsvector('english', body)` turns text into a **tsvector**: a sorted list of **lexemes**, which are words reduced to their stem, with common words like "the" and "a" dropped. "Running" and "runs" both become `run`, and "baking" becomes `bake`.

A search is a **tsquery**, reduced to lexemes the same way. `@@` asks whether a tsvector matches a tsquery, and `ts_rank` scores how well. **`websearch_to_tsquery`** reads queries the way people type them into a search box: `bread or cake`, `"exact phrase"`, and `-word` to exclude.

The table here stores its tsvector in a generated column, from the generated-columns lesson, so it's always up to date, and a **GIN index** on it makes searches fast.

**The limits.** Stemming follows rules, not a dictionary of every word: "running" becomes `run`, but irregular forms like "ran" stay as they are. It doesn't forgive typos either, so "databse" finds nothing. For typo-tolerant matching, use trigrams, from the index toolbox lesson. For one language it's usually all you need. Search across many languages, or at very large scale, is where dedicated search servers come in.

**Where it came from.** It started outside core, as the `tsearch2` contrib module by Oleg Bartunov and Teodor Sigaev. It was used so widely that it was built into Postgres 8.3 (2008). `websearch_to_tsquery` was added in Postgres 11 (2018).

**The Live View** shows table `articles`:

| Column | Meaning |
|---|---|
| `food` | A flag you'll set in the challenge |
| `body` | The article text |
| `search` | Its tsvector: each lexeme with the positions where it appears |

## Try it

See what Postgres keeps from a sentence. "The" and "was" are dropped, "runner" and "running" become different lexemes, and "ran" stays as it is.

```sql
SELECT to_tsvector('english', 'The runner was running and ran');
```

Search for "run" and rank the results. Article 1 matches through "Running". Article 2 ("runner") and article 6 ("ran") don't, because they're different lexemes.

```sql
SELECT id, body, ts_rank(search, q) AS rank
FROM articles, websearch_to_tsquery('english', 'run') q
WHERE search @@ q ORDER BY rank DESC;
```

Search the way people type into a search box: `or` between words, and `-` to exclude one. This finds articles about a database or bread, but not sourdough.

```sql
SELECT id, body FROM articles WHERE search @@ websearch_to_tsquery('english', 'database or bread -sourdough');
```

## Challenge

Set `food` to true on every article about baking or bread, in any form of those words, and on no other article. Watch out for "breadth".

Hint: UPDATE ... SET food = true WHERE search @@ websearch_to_tsquery('english', ...), using or like the last step.

```solution
UPDATE articles SET food = true WHERE search @@ websearch_to_tsquery('english', 'bake or bread');
```

```check
-- Articles 3 and 4 only. LIKE '%bread%' would also catch "breadth".
SELECT array_agg(id ORDER BY id) = '{3,4}' FROM articles WHERE food;
```
