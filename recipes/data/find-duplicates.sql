-- Find duplicate values: each value that appears more than once, how many
-- times, and the ids of its rows.
--
-- Run this before adding a unique constraint, which fails while any
-- duplicates exist. Read-only.
--
-- Change: signups, email (or several columns), and id.
-- To ignore case, group by lower(email).

SELECT email, count(*) AS copies,
       array_agg(id ORDER BY id) AS ids
FROM signups
GROUP BY email
HAVING count(*) > 1
ORDER BY copies DESC, email;
