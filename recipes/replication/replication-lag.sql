-- How far behind each replica and subscriber is.
--
-- Run it on the primary for streaming replicas and logical subscriptions
-- that read from it. An empty result means nothing is connected. Lag in
-- bytes is how much WAL the replica still has to apply; lag in time is
-- how old the newest change it has applied is. Read-only.

SELECT application_name, client_addr, state, sync_state,
       pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn)) AS replay_lag_bytes,
       write_lag, flush_lag, replay_lag
FROM pg_stat_replication
ORDER BY pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn) DESC NULLS FIRST;

-- On a replica, run this instead: how old its newest applied change is.
-- SELECT now() - pg_last_xact_replay_timestamp() AS replica_delay;
