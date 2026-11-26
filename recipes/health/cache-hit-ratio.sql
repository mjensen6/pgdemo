-- How often reads are served from memory instead of disk.
--
-- A busy OLTP database usually sits above 99%. Much lower can mean
-- shared_buffers is too small for the working set, or that big scans are
-- pushing hot pages out. "Disk" here means outside shared_buffers: the
-- operating system's cache may still serve it from memory. Counted since
-- the last stats reset. Read-only.

SELECT datname AS database,
       round(100.0 * blks_hit / nullif(blks_hit + blks_read, 0), 2) AS hit_percent,
       blks_read AS reads_from_disk
FROM pg_stat_database
WHERE datname IS NOT NULL
ORDER BY blks_read DESC;

-- The tables that read the most from disk, in this database.
SELECT relname AS table_name,
       round(100.0 * heap_blks_hit / nullif(heap_blks_hit + heap_blks_read, 0), 2) AS table_hit_percent,
       round(100.0 * idx_blks_hit / nullif(idx_blks_hit + idx_blks_read, 0), 2) AS index_hit_percent,
       heap_blks_read + coalesce(idx_blks_read, 0) AS reads_from_disk
FROM pg_statio_user_tables
ORDER BY heap_blks_read + coalesce(idx_blks_read, 0) DESC
LIMIT 10;
