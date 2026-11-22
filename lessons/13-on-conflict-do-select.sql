-- 13 ON CONFLICT DO SELECT (PG19): get or create in one query
CREATE TABLE tags (id serial PRIMARY KEY, name text UNIQUE);
INSERT INTO tags (name) VALUES ('postgres') ON CONFLICT (name) DO SELECT RETURNING id, name;
-- Same name again: no new row, and you still get the existing one back.
INSERT INTO tags (name) VALUES ('postgres') ON CONFLICT (name) DO SELECT RETURNING id, name;
SELECT count(*) FROM tags;
