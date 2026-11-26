-- Gaps and islands: find runs of consecutive days, such as login streaks.
--
-- In a run of consecutive days, the day minus its position in the list
-- stays the same, so that difference labels each run. The same trick works
-- for consecutive integers. Read-only.
--
-- Change: logins, user_name, and day.

WITH numbered AS (
  SELECT DISTINCT user_name, day FROM logins
), grouped AS (
  SELECT user_name, day,
         day - (row_number() OVER (PARTITION BY user_name ORDER BY day))::int AS run
  FROM numbered
)
SELECT user_name, min(day) AS first_day, max(day) AS last_day, count(*) AS days
FROM grouped
GROUP BY user_name, run
ORDER BY user_name, first_day;
