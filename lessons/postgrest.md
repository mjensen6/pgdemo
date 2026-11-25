---
title: PostgREST
tldr: Turn tables into a REST API with no code. Postgres permissions decide what the API allows.
---

```setup
DO $$ BEGIN CREATE ROLE web_anon NOLOGIN; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
CREATE SCHEMA api;
CREATE TABLE api.todos (id serial PRIMARY KEY, task text NOT NULL, done bool NOT NULL DEFAULT false);
INSERT INTO api.todos (task, done) VALUES ('buy milk', true), ('learn postgres', false), ('walk the dog', false);
CREATE TABLE api.projects (id int PRIMARY KEY, name text);
INSERT INTO api.projects VALUES (1, 'pgdemo'), (2, 'garden');
GRANT USAGE ON SCHEMA api TO web_anon;
GRANT SELECT ON api.todos TO web_anon;
-- Fetches a URL from PostgREST, for the Live View and the check.
CREATE FUNCTION api_get(path text) RETURNS text LANGUAGE plpgsql AS $f$
DECLARE body text;
BEGIN
  CREATE TEMP TABLE IF NOT EXISTS api_response (line text);
  TRUNCATE api_response;
  EXECUTE format('COPY api_response FROM PROGRAM %L',
                 'curl -s --max-time 2 http://127.0.0.1:3000' || path || ' | tr -d ''\n''; echo');
  SELECT line INTO body FROM api_response;
  RETURN coalesce(nullif(body, ''), '(PostgREST is not running)');
EXCEPTION WHEN others THEN RETURN '(PostgREST is not running)';
END $f$;
```

```view
SELECT 'GET /todos' AS request, api_get('/todos?select=id,task,done') AS response
UNION ALL
SELECT 'GET /projects', api_get('/projects');
```

```cleanup
\! kill "$(cat /tmp/pgdemo-postgrest.pid 2>/dev/null)" 2>/dev/null; rm -f /tmp/pgdemo-postgrest.pid /tmp/pgdemo-postgrest.conf
```

A lot of backend code just moves rows between a database and HTTP: list the todos, add one, filter by status, check the user is allowed. **PostgREST** is a server that does that for you. Point it at a schema, and every table and view in it becomes a REST endpoint:

| Request | Does |
|---|---|
| `GET /todos` | `SELECT * FROM api.todos`, as JSON |
| `GET /todos?done=is.false&select=task` | Filter and pick columns |
| `GET /todos?order=id.desc&limit=2` | Sort and limit |
| `POST /todos` with JSON | `INSERT` |
| `PATCH /todos?id=eq.2` and `DELETE /todos?id=eq.2` | `UPDATE` and `DELETE` |

**Postgres decides what's allowed.** PostgREST runs each request as a database role. Anonymous requests use the role named by `db-anon-role`, here `web_anon`. If that role lacks a privilege, Postgres refuses the query, and PostgREST returns the error as JSON. So `GRANT` and `REVOKE` are your API's access control, and row-level security (lesson 23) can limit which rows each user sees. Logged-in users send a JSON Web Token, and PostgREST switches to the role named in it.

**The schema cache.** PostgREST reads the list of tables when it starts. After creating a new table or column, tell it to reload with `NOTIFY pgrst, 'reload schema'`, which uses LISTEN and NOTIFY from lesson 20. Privilege changes need no reload, since Postgres checks those on every query.

**When it fits.** CRUD apps, internal tools, and the backend of a single-page or mobile app. Logic that doesn't fit a table goes into SQL functions, which become `POST /rpc/function_name`. Complex workflows that call other services still belong in an app server.

**Where it came from.** Joe Nelson started PostgREST in the mid-2010s, and it's written in Haskell. It's a separate open-source program, not part of Postgres. It powers the database API of Supabase, which made it widely known.

This lesson starts PostgREST inside the course container, on port 3000, serving the schema `api` from this playground. Resetting the lesson stops it.

**The Live View** asks the API for `/todos` and `/projects` every second and shows the JSON it returns, or its error.

## Try it

Write a PostgREST config and start it in the background. Then ask for the todos. The Live View fills in too.

```sh
[ -f /tmp/pgdemo-postgrest.pid ] && kill "$(cat /tmp/pgdemo-postgrest.pid)" 2>/dev/null; sleep 0.3
cat > /tmp/pgdemo-postgrest.conf <<CONF
db-uri = "postgres://postgres@/lesson_postgrest?host=/var/run/postgresql"
db-schemas = "api"
db-anon-role = "web_anon"
server-host = "127.0.0.1"
server-port = 3000
CONF
nohup postgrest /tmp/pgdemo-postgrest.conf >/tmp/pgdemo-postgrest.log 2>&1 & echo $! > /tmp/pgdemo-postgrest.pid
sleep 2
curl -s localhost:3000/todos; echo
```

Filter in the URL: only todos not done, only the `task` column. Then sort newest first and take two.

```sh
curl -s "localhost:3000/todos?done=is.false&select=task"; echo
curl -s "localhost:3000/todos?order=id.desc&limit=2"; echo
```

Try to add a todo with `POST`. `web_anon` may only read, so Postgres refuses, and PostgREST passes on its error: `permission denied for table todos`.

```sh
curl -s -X POST localhost:3000/todos -H 'Content-Type: application/json' -d '{"task": "write docs"}'; echo
```

Let anonymous users add todos: grant `INSERT`, and use of the sequence that numbers them. The same `POST` now works, and the new todo appears in the Live View.

```sql
GRANT INSERT ON api.todos TO web_anon;
GRANT USAGE ON SEQUENCE api.todos_id_seq TO web_anon;
\! curl -s -X POST localhost:3000/todos -H 'Content-Type: application/json' -d '{"task": "write docs"}' -w 'HTTP %{http_code}\n'
```

## Challenge

`GET /projects` is refused in the Live View. Make the projects readable through the API, read-only, so they show up there.

Hint: Which privilege did the setup give web_anon on todos? Grant the same on api.projects.

```solution
GRANT SELECT ON api.projects TO web_anon;
```

```check
SELECT api_get('/projects') LIKE '%"name":"pgdemo"%'
   AND NOT has_table_privilege('web_anon', 'api.projects', 'INSERT');
```
