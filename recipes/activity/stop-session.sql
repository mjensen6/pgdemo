-- Stop a session: cancel its query, or end the whole connection.
--
-- Read-only as written: it lists candidates and the command for each.
-- Copy the command you want and run it.
--   pg_cancel_backend    stops the current query; the session stays.
--   pg_terminate_backend ends the connection and rolls back its
--                        transaction. Use it when cancel isn't enough,
--                        such as for a session idle in a transaction.
-- You can stop your own sessions. Others need superuser or the
-- pg_signal_backend role.

SELECT pid, usename AS user_name, state,
       now() - coalesce(xact_start, query_start) AS age,
       left(regexp_replace(query, '\s+', ' ', 'g'), 60) AS query,
       format('SELECT pg_cancel_backend(%s);', pid) AS cancel,
       format('SELECT pg_terminate_backend(%s);', pid) AS terminate
FROM pg_stat_activity
WHERE backend_type = 'client backend'
  AND pid <> pg_backend_pid()
  AND state <> 'idle'
ORDER BY coalesce(xact_start, query_start);
