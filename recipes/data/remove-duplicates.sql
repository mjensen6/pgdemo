-- Remove duplicate rows, keeping the oldest of each.
--
-- WRITES: deletes rows. It runs in a transaction and shows what it would
-- delete, then rolls back. When the list is right, change ROLLBACK to
-- COMMIT and run it again.
--
-- Change: signups, email, and the ORDER BY that picks which copy to keep
-- (here the first created, with id breaking ties).

BEGIN;

WITH ranked AS (
  SELECT id, row_number() OVER (PARTITION BY email ORDER BY created_at, id) AS copy
  FROM signups
)
DELETE FROM signups s
USING ranked r
WHERE s.id = r.id AND r.copy > 1
RETURNING s.*;

ROLLBACK;  -- change to COMMIT
