-- Sessions that opened a transaction and then stopped doing anything.
--
-- An open transaction keeps its locks, and stops vacuum from removing
-- any row that changed since it began, so one forgotten session can bloat
-- every table. Usually an app that forgot to commit, or a person who left
-- a psql session at BEGIN. Read-only.
--
-- To end such sessions automatically, set
-- idle_in_transaction_session_timeout, for example to '10min'.
--
-- Change: the interval.

SELECT pid, usename AS user_name, datname AS database, application_name,
       now() - xact_start AS transaction_age,
       now() - state_change AS idle_for,
       left(regexp_replace(query, '\s+', ' ', 'g'), 100) AS last_query
FROM pg_stat_activity
WHERE state IN ('idle in transaction', 'idle in transaction (aborted)')
  AND now() - state_change > interval '1 minute'
ORDER BY state_change;
