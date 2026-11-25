#!/usr/bin/env bash
# Phase 2 acceptance test: the tools. Drives the real interface through
# tmux like phase1.sh, but keeps the container, image, and your progress.
# It does open and close the connect port, and answers no to uninstall.
set -euo pipefail
cd "$(dirname "$0")/.."

T="tmux -L pgdemo-test"
IN="docker compose exec -T -u postgres db tmux -L pgdemo"

screen() { $T capture-pane -p -t t 2>/dev/null || true; }
ok()     { printf 'ok   %s\n' "$*"; }
fail()   { printf 'FAIL %s\n\n--- screen ---\n' "$*"; screen; exit 1; }
wait_for() { # wait_for TEXT [SECONDS]
  local i
  for ((i = 0; i < ${2:-30} * 4; i++)); do
    screen | grep -qF -- "$1" && return 0
    sleep 0.25
  done
  fail "timed out waiting for: $1"
}
launch() { # launch [pgdemo args...]
  $T kill-server 2>/dev/null || true
  $IN kill-server 2>/dev/null || true
  $T new-session -d -s t -x 160 -y 45 "./pgdemo $*; echo PGDEMO-EXIT=\$?; sleep 3600"
}
key()   { $T send-keys -t t "$@"; }
click() { $T send-keys -t t -l $'\e[<0;'"$1;$2"'M'$'\e[<0;'"$1;$2"'m'; sleep 0.5; }
cleanup() {
  $T kill-server 2>/dev/null || true
  [[ -e compose.override.yaml ]] && ./pgdemo connect off >/dev/null 2>&1 || true
}
trap cleanup EXIT

./pgdemo setup >/dev/null

# help: every command in the spec is listed, and one command is explained.
h=$(./pgdemo help)
for c in list "section 8" "5 tldr" "5 info" "5 run" "5 view" "5 reset" "5 check" "5 hint" "5 solution" \
         check browse top psql connect setup doctor stop uninstall "help check"; do
  grep -qF "pgdemo $c " <<<"$h" || fail "pgdemo help does not list: pgdemo $c"
done
./pgdemo help check | grep -qF "Example:" || fail "pgdemo help check has no example"
! ./pgdemo help nope >/dev/null 2>&1 || fail "pgdemo help nope should fail"
grep -q "arrives in the next build" player/*.go pgdemo && fail "a command still says it arrives in the next build"
ok "help lists every command and explains one with an example"

# section: lists Sections, refuses an unbuilt one, opens a built one.
./pgdemo section | grep -qF "Keeping It Healthy" || fail "pgdemo section does not list the Sections"
out=$(./pgdemo section 6 2>&1 || true)  # Security and Testing: all new lessons, not built yet
grep -qF "coming soon" <<<"$out" || fail "pgdemo section 6 should say it is coming soon"
launch section 8
wait_for "Lesson 31: REPACK" 120
ok "pgdemo section 8 opens the Section's lesson"

# The Browse tab runs lazysql on the playground, and follows lesson changes.
click 12 45; wait_for "Browse: lesson_repack" 20; wait_for "public" 20
click 3 45; click 10 10
key p; wait_for "Lesson 27: pg_partman" 60
click 12 45; wait_for "Browse: lesson_pg_partman" 20; wait_for "public" 20
ok "the Browse tab shows lazysql and follows the lesson"
click 3 45; click 10 10
key q; wait_for "Quit and stop the database? y/n"; key y; wait_for "PGDEMO-EXIT=0" 60

# Standalone tools, each in its own terminal.
launch 1 view
wait_for "free_pct" 60
key q; wait_for "PGDEMO-EXIT=0" 20
ok "pgdemo 1 view shows the Live View"

launch top
wait_for "PostgreSQL 19" 60
key q; wait_for "PGDEMO-EXIT=" 20
ok "pgdemo top opens pg_activity"

launch browse 2
wait_for "lesson_transactional_ddl" 60
key q; wait_for "Exit LazySQL?" 10; key Enter; wait_for "PGDEMO-EXIT=0" 20
ok "pgdemo browse 2 opens lazysql on lesson 2"
$T kill-server 2>/dev/null || true

# connect: opens 127.0.0.1:5439 only, works for a real client, and closes.
./pgdemo connect 2 | grep -qF "postgresql://postgres@localhost:5439/lesson_transactional_ddl" \
  || fail "pgdemo connect 2 did not print the connection string"
[[ $(docker port pgdemo-db-1 5432) == "127.0.0.1:5439" ]] || fail "the connect port is not bound to 127.0.0.1 only"
docker run --rm --network host --entrypoint psql pgdemo:latest -X -Atc "SELECT 1" \
  postgresql://postgres@localhost:5439/lesson_transactional_ddl | grep -qx 1 || fail "a client could not connect through port 5439"
./pgdemo connect off >/dev/null
[[ -z $(docker port pgdemo-db-1 2>/dev/null) ]] || fail "pgdemo connect off left the port open"
ok "pgdemo connect opens 127.0.0.1:5439, a client connects, and off closes it"

# doctor: healthy now, and it spots a leftover replication slot.
./pgdemo doctor >/dev/null || fail "pgdemo doctor reported problems on a healthy setup"
docker compose exec -T -u postgres db psql -qc "SELECT pg_create_logical_replication_slot('pgdemo_test_slot', 'test_decoding')" >/dev/null
out=$(./pgdemo doctor 2>&1 || true)
docker compose exec -T -u postgres db psql -qc "SELECT pg_drop_replication_slot('pgdemo_test_slot')" >/dev/null
grep -qF "pg_drop_replication_slot('pgdemo_test_slot')" <<<"$out" || fail "pgdemo doctor missed a leftover replication slot"
ok "pgdemo doctor passes when healthy and explains a leftover slot"

# uninstall: asks first, and no means nothing is removed.
launch uninstall
wait_for "Remove everything? [y/N]" 30
key n Enter; wait_for "Nothing was removed." 10
docker image inspect pgdemo:latest >/dev/null 2>&1 || fail "answering no to uninstall still removed the image"
ok "pgdemo uninstall asks first, and no keeps everything"

echo "Phase 2 passed."
