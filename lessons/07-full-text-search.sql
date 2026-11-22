-- 07 Full-text search: built in since 8.3
CREATE TABLE articles (id serial, body text,
  search tsvector GENERATED ALWAYS AS (to_tsvector('english', body)) STORED);
INSERT INTO articles (body) VALUES
  ('Running a database on a home server'),
  ('The runner finished the marathon'),
  ('How to bake sourdough bread');
CREATE INDEX ON articles USING gin (search);
-- "run" finds "Running" because both stem to the same word. "runner" is a different word.
SELECT body, ts_rank(search, q) AS rank
FROM articles, websearch_to_tsquery('english', 'run') q WHERE search @@ q ORDER BY rank DESC;
-- Web-style search: quotes, OR, and minus all work.
SELECT body FROM articles WHERE search @@ websearch_to_tsquery('english', 'database or bread -sourdough');
