-- 12 MERGE (PG15): insert, update, and delete in one statement
CREATE TABLE stock (sku text PRIMARY KEY, qty int);
INSERT INTO stock VALUES ('apple', 5), ('pear', 2);
CREATE TABLE delivery (sku text, qty int);
INSERT INTO delivery VALUES ('apple', 10), ('pear', -2), ('plum', 7);
MERGE INTO stock s USING delivery d ON s.sku = d.sku
WHEN MATCHED AND s.qty + d.qty <= 0 THEN DELETE
WHEN MATCHED THEN UPDATE SET qty = s.qty + d.qty
WHEN NOT MATCHED THEN INSERT VALUES (d.sku, d.qty)
RETURNING merge_action(), s.*;
SELECT * FROM stock ORDER BY sku;
