# 21 Logical replication: copy live changes, now with sequences (PG19)
. "$(dirname "$0")/../lib/helpers.sh"
psql -X -q -d postgres -c 'DROP DATABASE IF EXISTS lesson_copy WITH (FORCE)' -c 'CREATE DATABASE lesson_copy' 2>/dev/null
schema="CREATE TABLE notes (id serial PRIMARY KEY, body text)"
quiet "$schema"; quiet "$schema" lesson_copy
quiet "INSERT INTO notes (body) VALUES ('first')"
sql "CREATE PUBLICATION pub FOR ALL TABLES, ALL SEQUENCES"
sql "SELECT slot_name FROM pg_create_logical_replication_slot('sub', 'pgoutput')"
conn="dbname=lesson user=${PGUSER:-postgres} host=${PGHOST:-/var/run/postgresql} port=${PGPORT:-5432}"
sql "CREATE SUBSCRIPTION sub CONNECTION '$conn' PUBLICATION pub WITH (create_slot = false, slot_name = 'sub')" lesson_copy
quiet "INSERT INTO notes (body) VALUES ('second')"
for _ in $(seq 1 50); do [[ $(quiet "SELECT count(*) FROM notes" lesson_copy) == 2 ]] && break; sleep 0.1; done
say "Both rows arrived in the second database."
sql "SELECT * FROM notes" lesson_copy
say "Sync the sequence too, so a failover would not reuse IDs."
sql "ALTER SUBSCRIPTION sub REFRESH SEQUENCES" lesson_copy
for _ in $(seq 1 50); do [[ $(quiet "SELECT last_value FROM notes_id_seq" lesson_copy) == 2 ]] && break; sleep 0.1; done
sql "SELECT last_value FROM notes_id_seq" lesson_copy
quiet "DROP SUBSCRIPTION sub" lesson_copy
psql -X -q -d postgres -c 'DROP DATABASE lesson_copy WITH (FORCE)'
