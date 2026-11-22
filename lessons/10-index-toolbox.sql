-- 10 Index toolbox: partial, expression, BRIN, trigram, exclusion
CREATE EXTENSION pg_trgm;
CREATE EXTENSION btree_gist;
CREATE TABLE users AS SELECT i AS id, 'User' || i || '@Mail.com' AS email,
  i % 100 = 0 AS active, now() - i * interval '1 minute' AS created FROM generate_series(1, 100000) i;

-- Partial: index only the rows you query.
CREATE INDEX users_active ON users (id) WHERE active;
-- Expression: index a computed value.
CREATE INDEX users_lower ON users (lower(email));
-- BRIN: tiny index for naturally ordered data.
CREATE INDEX users_brin ON users USING brin (created);
-- Trigram: makes LIKE '%middle%' indexable.
CREATE INDEX users_trgm ON users USING gin (email gin_trgm_ops);
ANALYZE users;
SELECT indexrelname AS index, pg_size_pretty(pg_relation_size(indexrelid)) AS size
FROM pg_stat_user_indexes WHERE relname = 'users' ORDER BY pg_relation_size(indexrelid);
EXPLAIN (COSTS OFF) SELECT id FROM users WHERE lower(email) = 'user5@mail.com';
EXPLAIN (COSTS OFF) SELECT id FROM users WHERE email LIKE '%er4242@%';

-- Exclusion constraint: the database itself refuses double bookings.
CREATE TABLE bookings (room int, during tstzrange, EXCLUDE USING gist (room WITH =, during WITH &&));
INSERT INTO bookings VALUES (1, '[2030-01-01 09:00, 2030-01-01 10:00)');
-- This next booking overlaps, so Postgres rejects it.
\set ON_ERROR_STOP off
INSERT INTO bookings VALUES (1, '[2030-01-01 09:30, 2030-01-01 11:00)');
\set ON_ERROR_STOP on
