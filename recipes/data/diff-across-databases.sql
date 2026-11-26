-- Diff a table against the same table in another database.
--
-- postgres_fdw maps the remote table in as a foreign table, then a normal
-- diff runs on it. The other database can be on another server.
--
-- Creates the postgres_fdw extension, a server called remote_db, and a
-- schema called remote. It drops and recreates that schema each run, so
-- keep your own tables out of it.
--
-- Change: dbname (and host, port for another server), the user mapping,
-- the table names, and id.
-- A superuser may connect with no password. Anyone else needs:
--   CREATE USER MAPPING FOR CURRENT_USER SERVER remote_db
--     OPTIONS (user 'reader', password '...');

CREATE EXTENSION IF NOT EXISTS postgres_fdw;

DROP SERVER IF EXISTS remote_db CASCADE;
CREATE SERVER remote_db FOREIGN DATA WRAPPER postgres_fdw
  OPTIONS (dbname 'recipes_remote');  -- add: host 'db2.internal', port '5432'
CREATE USER MAPPING FOR CURRENT_USER SERVER remote_db;

DROP SCHEMA IF EXISTS remote CASCADE;
CREATE SCHEMA remote;
IMPORT FOREIGN SCHEMA public LIMIT TO (customers) FROM SERVER remote_db INTO remote;

-- The local table may have extra columns, so compare only the shared ones.
SELECT coalesce(l.id, r.id) AS id,
       CASE WHEN r.id IS NULL THEN 'only here'
            WHEN l.id IS NULL THEN 'only in remote'
            ELSE 'different' END AS change,
       to_jsonb(l) - 'referrer' AS here,
       to_jsonb(r) AS remote
FROM customers_new l
FULL JOIN remote.customers r ON r.id = l.id
WHERE (l.id, l.email, l.name, l.plan) IS DISTINCT FROM (r.id, r.email, r.name, r.plan)
ORDER BY 1;
