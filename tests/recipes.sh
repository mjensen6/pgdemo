#!/usr/bin/env bash
# Recipes test: loads recipes/sample-data.sql into a throwaway container,
# runs every recipe, and checks each output for what it should find.
# Pass -v to print every recipe's output.
#
# It uses its own container from the pgdemo:latest image, with no volume,
# so your course, progress, and playgrounds are never touched. Build or
# pull the image first, by running ./pgdemo or tests/course.sh.
set -euo pipefail
cd "$(dirname "$0")/.."

verbose=false
[[ ${1:-} == -v ]] && verbose=true

image=${PGDEMO_IMAGE:-pgdemo:latest}
c=pgdemo-recipes-test-$$
fail() { echo "FAIL: $*" >&2; exit 1; }
trap 'docker rm -f "$c" >/dev/null 2>&1 || true' EXIT

# What each recipe's output must contain. Every recipe needs an entry, so
# a new recipe can't go untested.
declare -A expect=(
  [data/diff-two-tables]='(6 rows)|removed|added|changed'
  [data/diff-which-columns]='(3 rows)|ken@old.example.com|"pro"'
  [data/diff-except]='(9 rows)|only in old|only in new'
  [data/diff-across-databases]='(2 rows)|only here|different'
  [data/checksum-blocks]='(1 row)|120000'
  [data/find-duplicates]='(2 rows)|{1,3,5}'
  [data/remove-duplicates]='DELETE 3|ROLLBACK'
  [data/exact-row-counts]='events_a|200000'
  [data/gaps-and-islands]='(4 rows)|2026-03-04'
  [data/keyset-pagination]='(5 rows)'
  [data/batched-update]='updated 10000 rows'
  [data/batched-delete]='deleted'
  [health/table-sizes]='events_a'
  [health/bloat-estimate]='bloat_demo'
  [health/unused-indexes]='counting_since|customers_new_name'
  [health/duplicate-indexes]='{events_a_kind,events_a_kind_again}'
  [health/fk-without-index]='orders_customer_id_fkey|CREATE INDEX CONCURRENTLY ON orders (customer_id);'
  [health/cache-hit-ratio]='recipes_demo|table_hit_percent'
  [health/xid-age]='recipes_demo|percent_of_shutdown'
  [health/sequences-running-out]='tickets.id|93.1'
  [health/slowest-queries]='INSERT INTO events_a'
  [activity/long-running-queries]='pg_sleep'
  [activity/idle-in-transaction]='(1 row)|SELECT 1'
  [activity/blocking-locks]='SELECT count(*) FROM logins|LOCK TABLE logins'
  [activity/stop-session]='pg_terminate_backend('
  [schema/mostly-null-columns]='referrer'
  [schema/tables-without-primary-key]='logins'
  [schema/list-constraints]='FOREIGN KEY (customer_id) REFERENCES customers_new(id)'
  [schema/table-privileges]='report_reader|SELECT'
  [replication/replication-lag]='(0 rows)'
  [replication/slots-holding-wal]='forgotten_slot'
)

# Every recipe file has a test, and is listed in the README.
for f in recipes/*/*.sql; do
  name=${f#recipes/}; name=${name%.sql}
  [[ -v expect[$name] ]] || fail "$f has no entry in tests/recipes.sh"
  grep -qF "($name.sql)" recipes/README.md || fail "$f is not listed in recipes/README.md"
done
for name in "${!expect[@]}"; do
  [[ -f recipes/$name.sql ]] || fail "tests/recipes.sh tests recipes/$name.sql, which does not exist"
done

docker run -d --rm --name "$c" \
  -e POSTGRES_HOST_AUTH_METHOD=trust -e PGUSER=postgres \
  -v "$PWD/recipes:/recipes:ro" "$image" \
  postgres -c wal_level=logical -c shared_preload_libraries=pg_stat_statements >/dev/null
# The image's first start runs a temporary server, then restarts for real.
for _ in $(seq 120); do
  docker logs "$c" 2>&1 | grep -q 'PostgreSQL init process complete' &&
    docker exec "$c" pg_isready -q && break
  sleep 0.5
done
docker exec "$c" pg_isready -q || fail "the test server did not start"

psql() { docker exec -i "$c" psql -X -v ON_ERROR_STOP=1 -P pager=off "$@"; }

psql -q -f /recipes/sample-data.sql >/dev/null 2>&1 || {
  psql -f /recipes/sample-data.sql; fail "sample-data.sql failed"; }
echo "ok  sample data loaded"

# Some work for the activity and replication recipes to find: a long
# query holding a lock, a query waiting on it, a session idle inside a
# transaction, and a slot that nobody reads.
# The slot comes first: creating one waits for open transactions to end.
psql -q -d recipes_demo -c "SELECT pg_create_logical_replication_slot('forgotten_slot', 'test_decoding')" >/dev/null
docker exec -d "$c" psql -X -d recipes_demo -c "BEGIN; LOCK TABLE logins; SELECT pg_sleep(60); COMMIT;"
sleep 1
docker exec -d "$c" psql -X -d recipes_demo -c "SELECT count(*) FROM logins"
printf 'BEGIN;\nSELECT 1;\n\\! sleep 60\n' | docker exec -i "$c" sh -c 'cat > /tmp/idle.sql'
docker exec -d "$c" psql -X -d recipes_demo -f /tmp/idle.sql
sleep 2

# The activity recipes look for work older than a minute. Here the work
# is a few seconds old, so the test shortens the interval.
# The activity recipes run first, then the test ends those sessions, so
# their lock can't hold up the recipes that read every table.
for name in $(printf '%s\n' "${!expect[@]}" | sort); do
  if [[ $name != activity/* && -z ${ended:-} ]]; then
    psql -q -d recipes_demo -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity
      WHERE backend_type = 'client backend' AND pid <> pg_backend_pid()" >/dev/null
    ended=1
  fi
  out=$(sed "s/interval '1 minute'/interval '1 second'/" "recipes/$name.sql" |
        psql -d recipes_demo -f - 2>&1) || { echo "$out"; fail "$name.sql failed"; }
  $verbose && printf '\n== %s ==\n%s\n' "$name" "$out"
  IFS='|' read -ra wants <<<"${expect[$name]}"
  for want in "${wants[@]}"; do
    grep -qF -- "$want" <<<"$out" || { echo "$out"; fail "$name.sql output lacks: $want"; }
  done
  echo "ok  $name"
done
echo "All ${#expect[@]} recipes pass."
