-- Sample data for trying the recipes.
--
-- Creates the database recipes_demo, plus recipes_remote for the
-- cross-database diff, and fills them with small tables that each recipe
-- has something to find. Running it again starts over.
--
--   psql -X -f recipes/sample-data.sql
--
-- Then run any recipe in recipes_demo:
--
--   psql -X -d recipes_demo -f recipes/data/diff-two-tables.sql

DROP DATABASE IF EXISTS recipes_demo WITH (FORCE);
DROP DATABASE IF EXISTS recipes_remote WITH (FORCE);
CREATE DATABASE recipes_demo;
CREATE DATABASE recipes_remote;

\connect recipes_remote

-- The same customers as customers_new in recipes_demo, except two rows.
CREATE TABLE customers (id int PRIMARY KEY, email text, name text, plan text);
INSERT INTO customers VALUES
  (1, 'ada@example.com',     'Ada',     'pro'),
  (2, 'grace@example.com',   'Grace',   'free'),
  (4, 'edsger@example.com',  'Edsger',  'pro'),
  (5, 'barbara@example.com', 'Barbara', 'pro'),     -- free in recipes_demo
  (6, 'ken@example.com',     'Ken',     'free'),
  (7, 'frances@example.com', 'Frances', 'team');
  -- 8 linus is missing here

\connect recipes_demo

CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Two versions of one table, for the diff recipes.
-- Compared with customers_old, customers_new has:
--   removed: 3 alan
--   added:   7 frances, 8 linus
--   changed: 2 grace (plan), 5 barbara (plan, from NULL), 6 ken (email)
CREATE TABLE customers_old (id int PRIMARY KEY, email text, name text, plan text);
CREATE TABLE customers_new (id int PRIMARY KEY, email text, name text, plan text, referrer text);
INSERT INTO customers_old VALUES
  (1, 'ada@example.com',     'Ada',     'pro'),
  (2, 'grace@example.com',   'Grace',   'pro'),
  (3, 'alan@example.com',    'Alan',    'free'),
  (4, 'edsger@example.com',  'Edsger',  'pro'),
  (5, 'barbara@example.com', 'Barbara', NULL),
  (6, 'ken@old.example.com', 'Ken',     'free');
INSERT INTO customers_new (id, email, name, plan) VALUES
  (1, 'ada@example.com',     'Ada',     'pro'),
  (2, 'grace@example.com',   'Grace',   'free'),
  (4, 'edsger@example.com',  'Edsger',  'pro'),
  (5, 'barbara@example.com', 'Barbara', 'free'),
  (6, 'ken@example.com',     'Ken',     'free'),
  (7, 'frances@example.com', 'Frances', 'team'),
  (8, 'linus@example.com',   'Linus',   'free');
-- referrer is a new column nobody has filled in yet, for the mostly-NULL
-- recipe.
-- An index nothing uses, for the unused-indexes recipe.
CREATE INDEX customers_new_name ON customers_new (name);

-- A big table copied twice, with one row changed in the copy, for the
-- block checksum recipe.
CREATE TABLE events_a (id bigint PRIMARY KEY, kind text, payload text);
INSERT INTO events_a SELECT i, (ARRAY['view','click','buy'])[1 + i % 3], md5(i::text)
FROM generate_series(1, 200000) i;
CREATE TABLE events_b AS SELECT * FROM events_a;
ALTER TABLE events_b ADD PRIMARY KEY (id);
UPDATE events_b SET kind = 'refund' WHERE id = 123456;
-- Two identical indexes, for the duplicate-indexes recipe.
CREATE INDEX events_a_kind ON events_a (kind);
CREATE INDEX events_a_kind_again ON events_a (kind);

-- Signups with duplicate emails, for the duplicate recipes.
CREATE TABLE signups (id serial PRIMARY KEY, email text NOT NULL, created_at timestamptz NOT NULL);
INSERT INTO signups (email, created_at) VALUES
  ('ada@example.com',   '2026-01-01'),
  ('grace@example.com', '2026-01-02'),
  ('ada@example.com',   '2026-01-05'),
  ('ken@example.com',   '2026-01-06'),
  ('ada@example.com',   '2026-01-09'),
  ('grace@example.com', '2026-01-10');

-- Orders that point at customers with no index on the foreign key.
CREATE TABLE orders (
  id serial PRIMARY KEY,
  customer_id int NOT NULL REFERENCES customers_new,
  total numeric NOT NULL,
  status text
);
INSERT INTO orders (customer_id, total)
SELECT (ARRAY[1, 2, 4, 5, 6, 7, 8])[1 + i % 7], round((random() * 100)::numeric, 2) FROM generate_series(1, 50000) i;

-- Daily logins with no primary key, for gaps-and-islands and the
-- missing-primary-key recipe. Ada has a 4-day run, then a 2-day run.
CREATE TABLE logins (user_name text, day date);
INSERT INTO logins VALUES
  ('ada', '2026-03-01'), ('ada', '2026-03-02'), ('ada', '2026-03-03'), ('ada', '2026-03-04'),
  ('ada', '2026-03-08'), ('ada', '2026-03-09'),
  ('ken', '2026-03-02'), ('ken', '2026-03-05');

-- A serial column that has used 93% of its int range.
CREATE TABLE tickets (id serial PRIMARY KEY, subject text);
SELECT setval('tickets_id_seq', 2000000000);
INSERT INTO tickets (subject) VALUES ('printer on fire');

-- A table that's half dead rows, for the bloat recipe. Autovacuum is off
-- for it, so the dead rows stay.
CREATE TABLE bloat_demo (id int, filler text) WITH (autovacuum_enabled = off);
INSERT INTO bloat_demo SELECT i, repeat('x', 200) FROM generate_series(1, 100000) i;
DELETE FROM bloat_demo WHERE id % 2 = 0;

-- A role with read access to one table, for the privileges recipe.
DO $$ BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'report_reader') THEN
    CREATE ROLE report_reader;
  END IF;
END $$;
GRANT SELECT ON customers_new TO report_reader;

-- Planner statistics, which the mostly-NULL recipe reads.
ANALYZE customers_new, customers_old, events_a, events_b, signups, orders, logins, tickets;
