-- Replication slots, and how much WAL each one keeps on disk.
--
-- A slot makes the server keep WAL until its consumer reads it. If the
-- consumer goes away, the slot keeps WAL forever and the disk fills.
-- Watch inactive slots with a growing retained size. Setting
-- max_slot_wal_keep_size caps it: the slot is invalidated instead of the
-- disk filling up. Read-only.
--
-- To drop a slot that's no longer needed:
--   SELECT pg_drop_replication_slot('slot_name');

SELECT slot_name, slot_type, database, active,
       pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), restart_lsn)) AS wal_retained,
       wal_status,  -- reserved, extended, unreserved, or lost
       inactive_since  -- Postgres 17 and later
FROM pg_replication_slots
ORDER BY pg_wal_lsn_diff(pg_current_wal_lsn(), restart_lsn) DESC NULLS LAST;
