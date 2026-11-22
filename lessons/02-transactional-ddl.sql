-- 02 Transactional DDL: undo a schema change
CREATE TABLE users (id int, name text);
INSERT INTO users VALUES (1, 'Ada');
BEGIN;
ALTER TABLE users DROP COLUMN name;
CREATE TABLE oops (id int);
SELECT * FROM users;
-- Changed our mind. Everything comes back.
ROLLBACK;
SELECT * FROM users;
SELECT to_regclass('oops') AS oops_table;
