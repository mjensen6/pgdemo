-- 09 Generated columns: virtual by default (PG18)
CREATE TABLE line_items (
  price numeric, qty int,
  total numeric GENERATED ALWAYS AS (price * qty) VIRTUAL,   -- computed on read
  taxed numeric GENERATED ALWAYS AS (price * qty * 1.08) STORED -- saved to disk
);
INSERT INTO line_items VALUES (10, 3);
SELECT * FROM line_items;
UPDATE line_items SET qty = 5;
SELECT * FROM line_items;
