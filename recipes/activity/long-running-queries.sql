-- Queries that have been running longer than a minute.
--
-- Only queries running now: idle-in-transaction.sql finds sessions that
-- are holding a transaction open between queries. Read-only. To stop one,
-- see stop-session.sql.
--
-- Change: the interval.

SELECT pid, usename AS user_name, datname AS database,
       now() - query_start AS running_for,
       wait_event_type, wait_event,
       left(regexp_replace(query, '\s+', ' ', 'g'), 100) AS query
FROM pg_stat_activity
WHERE state = 'active'
  AND backend_type = 'client backend'
  AND pid <> pg_backend_pid()
  AND now() - query_start > interval '1 minute'
ORDER BY query_start;
