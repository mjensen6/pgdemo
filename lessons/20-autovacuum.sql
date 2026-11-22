-- 20 Autovacuum insight (PG19): see what it will clean next
CREATE TABLE busy (id int) WITH (autovacuum_enabled = false);
INSERT INTO busy SELECT generate_series(1, 50000);
DELETE FROM busy WHERE id % 2 = 0;
SELECT pg_stat_force_next_flush();
SELECT pg_sleep(0.2);
\x on
SELECT * FROM pg_stat_autovacuum_scores WHERE relname = 'busy';
\x off
-- New in 19: autovacuum can use parallel workers. The server limit is 0 (off) by default.
SHOW autovacuum_max_parallel_workers;
ALTER TABLE busy SET (autovacuum_parallel_workers = 2);
SELECT reloptions FROM pg_class WHERE relname = 'busy';
