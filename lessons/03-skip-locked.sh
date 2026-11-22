# 03 SKIP LOCKED: a job queue with no extra software
. "$(dirname "$0")/../lib/helpers.sh"
quiet "CREATE TABLE jobs (id int PRIMARY KEY, done bool DEFAULT false); INSERT INTO jobs SELECT generate_series(1, 6)"
say "Worker A claims jobs 1-2 and holds them for 2 seconds."
psql -X -q -At -d lesson -c "BEGIN" -c "SELECT id FROM jobs ORDER BY id LIMIT 2 FOR UPDATE" -c "SELECT pg_sleep(2)" -c "COMMIT" >/dev/null &
sleep 0.5
say "Worker B skips the locked jobs instead of waiting."
sql "SELECT id FROM jobs ORDER BY id LIMIT 2 FOR UPDATE SKIP LOCKED"
wait
