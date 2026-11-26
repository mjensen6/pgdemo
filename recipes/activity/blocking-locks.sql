-- Who is waiting on a lock, and which session is blocking them.
--
-- pg_blocking_pids does the hard part. The blocker at the root of a chain
-- is the one to look at: often a long transaction or an idle one. Read-only.

SELECT w.pid AS waiting_pid,
       now() - w.query_start AS waiting_for,
       left(regexp_replace(w.query, '\s+', ' ', 'g'), 60) AS waiting_query,
       b.pid AS blocking_pid,
       b.state AS blocker_state,
       now() - b.xact_start AS blocker_transaction_age,
       left(regexp_replace(b.query, '\s+', ' ', 'g'), 60) AS blocker_query
FROM pg_stat_activity w
CROSS JOIN LATERAL unnest(pg_blocking_pids(w.pid)) AS blocker(pid)
JOIN pg_stat_activity b ON b.pid = blocker.pid
ORDER BY w.query_start;
