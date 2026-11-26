-- Keyset pagination: fetch the next page after the last row you saw.
--
-- OFFSET 100000 makes Postgres read and throw away 100,000 rows, so pages
-- get slower the deeper you go. Keyset pagination remembers the last row's
-- key and starts there, so every page costs the same. It needs an index
-- that matches the ORDER BY, here on (kind, id). Read-only.
--
-- Change: events_a, the sort columns, and the last row's values.

-- First page.
SELECT id, kind FROM events_a
ORDER BY kind, id
LIMIT 5;

-- Next page: pass the last row of the page before. The row comparison
-- (kind, id) > (...) matches the two-column ORDER BY exactly.
SELECT id, kind FROM events_a
WHERE (kind, id) > ('buy', 14)
ORDER BY kind, id
LIMIT 5;
