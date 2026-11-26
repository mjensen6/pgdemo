-- How close each database and table is to transaction ID wraparound.
--
-- Postgres counts transactions with 32-bit IDs. Vacuum "freezes" old rows
-- so their IDs can be reused. Autovacuum forces a freeze when a table
-- reaches autovacuum_freeze_max_age (200 million by default). If the oldest
-- row ever nears about 2 billion, Postgres stops new writes to protect the
-- data. Above 50% here deserves a look: often a long-open transaction or
-- an old replication slot is blocking the freeze. Read-only.

SELECT datname AS database,
       age(datfrozenxid) AS xid_age,
       round(100.0 * age(datfrozenxid) / current_setting('autovacuum_freeze_max_age')::bigint, 1)
         AS percent_of_forced_freeze,
       round(100.0 * age(datfrozenxid) / 2000000000, 1) AS percent_of_shutdown
FROM pg_database
ORDER BY age(datfrozenxid) DESC;

-- The oldest tables in this database. A table's TOAST table, which holds
-- its large values, is frozen separately, so the older of the two counts.
SELECT c.oid::regclass AS table_name,
       greatest(age(c.relfrozenxid), age(t.relfrozenxid)) AS xid_age,
       pg_size_pretty(pg_total_relation_size(c.oid)) AS size
FROM pg_class c
LEFT JOIN pg_class t ON t.oid = c.reltoastrelid
WHERE c.relkind IN ('r', 'm')
ORDER BY 2 DESC, pg_total_relation_size(c.oid) DESC
LIMIT 10;
