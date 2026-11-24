-- 15 pg_partman: create future partitions automatically
CREATE SCHEMA partman;
CREATE EXTENSION pg_partman SCHEMA partman;
CREATE TABLE metrics (at timestamptz NOT NULL, value int) PARTITION BY RANGE (at);
SELECT partman.create_parent(p_parent_table := 'public.metrics', p_control := 'at', p_interval := '1 month', p_premake := 2);
SELECT count(*) AS partitions FROM pg_inherits WHERE inhparent = 'metrics'::regclass;
-- Run this on a schedule (lesson 26) and partitions stay ahead of your data.
CALL partman.run_maintenance_proc();
SELECT inhrelid::regclass AS partition FROM pg_inherits WHERE inhparent = 'metrics'::regclass ORDER BY 1 LIMIT 4;
