#!/usr/bin/env bash
# Phase 1 acceptance test. Drives the real interface through tmux, starting
# from a clean state with no pgdemo container or volume. The image is kept
# unless you pass --rebuild.
#
# The whole run happens inside a tmux server of its own, so it also covers
# launching pgdemo from inside an existing tmux session.
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
  $T new-session -d -s t -x 160 -y 45 "./pgdemo $*; echo PGDEMO-EXIT=\$?; sleep 3600"
}
key()   { $T send-keys -t t "$@"; }
click() { $T send-keys -t t -l $'\e[<0;'"$1;$2"'M'$'\e[<0;'"$1;$2"'m'; sleep 0.5; }
focus() { $IN display-message -p -t pgdemo '#{window_name}.#{pane_index}'; }
type_sql() {
  $IN send-keys -t pgdemo:Lesson.1 -l "$1"
  $IN send-keys -t pgdemo:Lesson.1 Enter
}
quit() {
  key q; wait_for "Quit and stop the database? y/n"
  key y; wait_for "PGDEMO-EXIT=0" 60
  screen | grep -qF "Run pgdemo to pick up where you left off." || fail "q did not print how to resume"
  [[ -z $(docker compose ps --status running -q db) ]] || fail "q did not stop the database"
}
trap '$T kill-server 2>/dev/null || true' EXIT

if [[ ${1:-} == --rebuild ]]; then docker compose down -v --rmi local; else docker compose down -v; fi
rm -rf .pgdemo && mkdir -p .pgdemo && touch .pgdemo/path-asked  # leave the tester's PATH alone

# 1. Builds, starts, and opens lesson 1 in three panes with no manual steps.
launch
wait_for "Course Overview" 900
wait_for "Lesson pane"
key b; sleep 0.5; screen | grep -qF "Course Overview" || fail "b went back past the Course Overview"
key Enter
wait_for "Press Enter to run step 1." 120
key b; wait_for "Course Overview"
key Enter; wait_for "Press Enter to run step 1."
wait_for "lesson_mvcc_bloat=#"
wait_for "Live View"
ok "1. pgdemo opens lesson 1 in three panes"

# 2. Enter steps through, and the Live View numbers change.
key Enter
wait_for "UPDATE 10000"
wait_for "│ 10000 │ 10000 │"
ok "2. Enter runs a step and the Live View shows the dead rows"

# 3 and 4. The SQL pane is connected; c fails, then passes after solving.
key Enter; wait_for "VACUUM"
key Enter; wait_for "Your turn."
key c; wait_for "Not yet."
type_sql "REPACK t;"
wait_for "REPACK"
key c; wait_for "● Done!"
ok "3. the SQL pane is connected and takes the challenge"

# 9. Mouse through the nested tmux: click each pane and tab.
click 110 6;  [[ $(focus) == Lesson.1 ]] || fail "clicking the SQL pane did not select it"
click 110 35; [[ $(focus) == Lesson.2 ]] || fail "clicking the Live View did not select it"
click 19 45;  [[ $(focus) == Top.0 ]]    || fail "clicking the Top tab did not switch to it"
click 3 45;   [[ $(focus) == Lesson.1 || $(focus) == Lesson.2 ]] || fail "clicking the Lesson tab did not switch back"
click 10 10;  [[ $(focus) == Lesson.0 ]] || fail "clicking the Lesson pane did not select it"

quit
./pgdemo list | grep -q "●   1  mvcc-bloat" || fail "pgdemo list does not show lesson 1 as done"
./pgdemo list | grep -q "coming soon" || fail "pgdemo list does not show unbuilt lessons"
[[ $(./pgdemo list | grep -cE '^  [○◐●·] +[0-9]+') == 43 ]] || fail "pgdemo list does not show all 43 lessons"
ok "4. c fails before solving, passes after, and list shows ●"

# 8. Running pgdemo again restores progress and the playground.
launch
wait_for "Lesson 2: Transactional DDL" 120
./pgdemo 1 check | grep -qF "is done" || fail "lesson 1's playground lost the learner's REPACK"
ok "8. q stops and resumes, keeping progress and the playground"

# 5. One session across steps, so ROLLBACK works.
key Enter; wait_for "ALTER TABLE"
key Enter; wait_for "1 | Ada"
quit
ok "5. pgdemo 2 keeps one session, so ROLLBACK brings the column back"

# 6. REPACK shows the rebuild while the background writer keeps landing writes.
launch repack
wait_for "Press Enter to run step 1." 120
key Enter; wait_for "27 MB | 20000"
key Enter; wait_for "REPACK" 60
writes() { screen | sed -n 's/^│ writes so far: \([0-9][0-9]*\).*/\1/p'; }  # output lines only
for _ in {1..120}; do [[ $(writes | wc -l) == 2 ]] && break; sleep 0.25; done
before=$(writes | head -1); stopped=$(writes | tail -1)
[[ -n $before ]] || fail "the repack step printed no write counts"
(( stopped > before )) || fail "writes did not keep landing ($before then $stopped)"
quit
ok "6. pgdemo repack rebuilds while writes land ($before, then $stopped)"

# 7. tldr, info, reset, hint, and solution.
./pgdemo 1 tldr     | grep -qF "Updates leave dead rows behind" || fail "tldr"
./pgdemo 1 info     | grep -qF "Step 2"                        || fail "info"
./pgdemo 1 hint     | grep -qF "Hint:"                         || fail "hint"
./pgdemo 1 solution | grep -qF "REPACK t"                      || fail "solution"
./pgdemo 1 reset    | grep -qF "back at step 1"                || fail "reset"
./pgdemo 1 check    | grep -qF "Not yet"                       || fail "reset did not rebuild the playground"
./pgdemo list       | grep -q "●   1  mvcc-bloat"              || fail "reset removed the done status"
docker compose stop >/dev/null 2>&1
ok "7. tldr, info, hint, solution, and reset work, and reset keeps done"

ok "9. everything above ran from inside an existing tmux session"
echo "Phase 1 passed."
