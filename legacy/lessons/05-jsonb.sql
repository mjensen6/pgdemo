-- 05 JSONB: documents inside a relational database
CREATE TABLE products (id serial PRIMARY KEY, doc jsonb);
INSERT INTO products (doc) VALUES
  ('{"name": "Lamp", "tags": ["home"], "price": 30}'),
  ('{"name": "Tent", "tags": ["outdoor", "camping"], "price": 120}');
-- Pull out fields.
SELECT doc->>'name' AS name, (doc->>'price')::int AS price FROM products;
-- @> means "contains". A GIN index makes it fast on big tables.
CREATE INDEX ON products USING gin (doc);
SELECT doc->>'name' AS name FROM products WHERE doc @> '{"tags": ["camping"]}';
-- Update one field in place.
UPDATE products SET doc = jsonb_set(doc, '{price}', '99') WHERE doc->>'name' = 'Tent';
SELECT doc FROM products WHERE doc->>'name' = 'Tent';
