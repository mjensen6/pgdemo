# 26 pg_cron: scheduled jobs inside the database
. "$(dirname "$0")/../lib/helpers.sh"
psql -X -q -d postgres -c 'CREATE EXTENSION IF NOT EXISTS pg_cron' 2>/dev/null
quiet "CREATE TABLE ticks (at timestamptz DEFAULT now())"
sql "SELECT cron.schedule_in_database('tick', '1 second', 'INSERT INTO ticks DEFAULT VALUES', 'lesson') AS job_id" postgres
say "Waiting 4 seconds."
sleep 4
sql "SELECT cron.unschedule('tick')" postgres
sql "SELECT count(*) > 0 AS job_ran, count(*) AS ticks FROM ticks"
[[ $(quiet "SELECT count(*) FROM ticks") -gt 0 ]]
