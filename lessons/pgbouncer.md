---
title: PgBouncer
tldr: A connection pooler that lets thousands of app connections share a handful of real Postgres connections.
---

```setup
CREATE EXTENSION dblink;
CREATE TABLE t AS SELECT generate_series(1, 1000) AS i;
-- Reads one PgBouncer setting through its admin console, or nothing if it isn't running.
CREATE FUNCTION bouncer_setting(name text) RETURNS text LANGUAGE plpgsql AS $$
BEGIN
  RETURN (SELECT value FROM dblink('host=127.0.0.1 port=6439 dbname=pgbouncer user=postgres', 'SHOW CONFIG')
            AS c(key text, value text, dflt text, changeable text) WHERE key = name);
EXCEPTION WHEN others THEN RETURN NULL;
END $$;
```

```view
SELECT coalesce(nullif(application_name, ''), '(none)') AS client, count(*) AS real_connections
FROM pg_stat_activity
WHERE datname = current_database() AND pid <> pg_backend_pid() AND backend_type = 'client backend'
GROUP BY 1 ORDER BY 1;
```

```cleanup
\! kill "$(cat /tmp/pgdemo-pgbouncer/pgbouncer.pid 2>/dev/null)" 2>/dev/null; rm -rf /tmp/pgdemo-pgbouncer
```

Every connection to Postgres gets its own server **process**. Starting one takes a few milliseconds, and each uses some memory even when idle. The server allows `max_connections` of them, 100 by default. That's plenty for one app server, but not for 50 app servers with 20 workers each, or serverless functions that each open their own connection.

Most of those connections sit idle most of the time, between queries. A **connection pooler** sits between the app and Postgres. The app opens as many connections to the pooler as it likes, and the pooler shares a few real Postgres connections among them, handing one out only while it's needed.

**PgBouncer** is the most widely used pooler. Its **pool mode** decides how long a client keeps a real connection:

| Pool mode | A client keeps a real connection for... |
|---|---|
| `session` | Its whole session. Safe for everything, but pools the least |
| `transaction` | One transaction, then it goes back to the pool. The usual choice |
| `statement` | One statement. Rarely used |

**The catch with transaction mode.** Your next transaction may run on a different real connection, so anything tied to a session doesn't carry over: `SET`, temporary tables, `LISTEN`, and session-level advisory locks. Most apps are fine, and prepared statements have worked in transaction mode since PgBouncer 1.21 (2023).

PgBouncer has an **admin console**, a pretend database named `pgbouncer`. You connect to it with psql and type commands such as `SHOW POOLS`, `SHOW CONFIG`, or `SET default_pool_size = 10` to change a setting while it runs.

**Where it came from.** PgBouncer was written at Skype in 2007 to serve its huge number of connections, and it's a separate program, not part of Postgres. Postgres itself still has no built-in pooler, so PgBouncer, or a newer one like PgCat or Supavisor, runs in front of most busy Postgres servers.

This lesson starts PgBouncer inside the course container, listening on port 6439. It shares the whole server, so leaving the lesson stops it.

**The Live View** counts the real Postgres connections to this playground, grouped by the program that opened them. `pgcli` is your SQL pane and `psql` is the lesson's own session.

## Try it

Write a PgBouncer config and start it. It pools connections to this playground in transaction mode, with at most **3** real connections. Then connect through it, on port 6439, to prove it works.

```sh
d=/tmp/pgdemo-pgbouncer
[ -f $d/pgbouncer.pid ] && kill "$(cat $d/pgbouncer.pid)" 2>/dev/null; sleep 0.3
rm -rf $d; mkdir -p $d
cat > $d/pgbouncer.ini <<CONF
[databases]
lesson_pgbouncer = host=/var/run/postgresql dbname=lesson_pgbouncer user=postgres
[pgbouncer]
listen_addr = 127.0.0.1
listen_port = 6439
unix_socket_dir = $d
auth_type = any
admin_users = postgres
pool_mode = transaction
default_pool_size = 3
logfile = $d/pgbouncer.log
pidfile = $d/pgbouncer.pid
CONF
pgbouncer -d $d/pgbouncer.ini && sleep 0.5
psql -h 127.0.0.1 -p 6439 -d lesson_pgbouncer -Atc "SELECT 'Connected through PgBouncer'"
```

Now 40 clients hammer the playground through PgBouncer for 5 seconds, using pgbench, Postgres's benchmarking tool. Halfway through, the step asks PgBouncer's admin console how the pool looks: many clients (`cl_active`, `cl_waiting`), only 3 real connections (`sv_active`). Watch `pgbench` in the Live View: it never goes above 3.

```sh
pgbench -h 127.0.0.1 -p 6439 -n -c 40 -T 5 -f <(echo 'SELECT count(*) FROM t;') lesson_pgbouncer 2>&1 | grep -E 'number of clients|tps' &
sleep 2.5
psql -h 127.0.0.1 -p 6439 -d pgbouncer --csv -c "SHOW POOLS" | cut -d, -f1,3,4,7,16
wait
```

## Challenge

Three real connections made most of the 40 clients wait. Raise PgBouncer's pool size to **5** while it runs, through its admin console, from the SQL pane.

Hint: Use \! and the psql command from the last step that opens the admin console (-d pgbouncer), with -c and the SET from the intro.

```solution
\! psql -h 127.0.0.1 -p 6439 -d pgbouncer -c "SET default_pool_size = 5"
```

```check
SELECT coalesce(bouncer_setting('default_pool_size') = '5', false);
```
