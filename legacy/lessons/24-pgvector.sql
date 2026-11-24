-- 24 pgvector: similarity search for AI apps
CREATE EXTENSION vector;
-- Real embeddings have hundreds of dimensions. Three keeps it readable: [animal, food, tech].
CREATE TABLE docs (title text, embedding vector(3));
INSERT INTO docs VALUES
  ('Dog training tips', '[0.9, 0.1, 0.0]'),
  ('Best pizza dough',  '[0.0, 0.9, 0.1]'),
  ('Postgres indexing', '[0.0, 0.0, 1.0]'),
  ('Robot pet review',  '[0.6, 0.0, 0.7]');
CREATE INDEX ON docs USING hnsw (embedding vector_cosine_ops);
-- <=> is cosine distance. Smaller means more similar.
SELECT title, round((embedding <=> '[1, 0, 0.1]')::numeric, 3) AS distance
FROM docs ORDER BY embedding <=> '[1, 0, 0.1]' LIMIT 3;
