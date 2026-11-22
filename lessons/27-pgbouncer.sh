# 27 PgBouncer: many clients, few real connections
. "$(dirname "$0")/../lib/helpers.sh"
d=$(mktemp -d); sock=${PGHOST:-/var/run/postgresql}; port=${PGPORT:-5432}
cat > "$d/b.ini" <<C
[databases]
lesson = host=$sock port=$port dbname=lesson user=${PGUSER:-postgres}
[pgbouncer]
listen_addr = 127.0.0.1
listen_port = 6439
unix_socket_dir = $d
auth_type = any
admin_users = ${PGUSER:-postgres}
pool_mode = transaction
default_pool_size = 3
logfile = $d/b.log
pidfile = $d/b.pid
C
pgbouncer -d "$d/b.ini"
trap 'kill "$(cat "$d/b.pid")" 2>/dev/null; rm -rf "$d"' EXIT
sleep 0.3
quiet "CREATE TABLE t AS SELECT generate_series(1, 1000) i"
say "40 clients connect through PgBouncer for 3 seconds."
( pgbench -h 127.0.0.1 -p 6439 -n -c 40 -T 3 -f <(echo 'SELECT count(*) FROM t;') lesson 2>&1 | grep -E 'clients|tps' ) &
sleep 1.5
say "Real Postgres connections while 40 clients are active:"
sql "SELECT count(*) AS real_connections FROM pg_stat_activity WHERE datname = 'lesson' AND pid <> pg_backend_pid() AND application_name <> 'psql'"
wait
