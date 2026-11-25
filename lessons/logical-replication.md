---
title: Logical replication, now with sequences
tldr: Stream row changes from one database to another as they happen. Postgres 19 can copy sequences too, so a switchover doesn't reuse IDs.
---

```setup
DROP DATABASE IF EXISTS logical_copy WITH (FORCE);
CREATE DATABASE logical_copy;
CREATE EXTENSION dblink;
CREATE TABLE notes (id serial PRIMARY KEY, body text);
INSERT INTO notes (body) VALUES ('first');
-- The check waits briefly for changes to arrive in the copy.
CREATE FUNCTION copy_has(want_rows int, want_seq int) RETURNS bool LANGUAGE plpgsql AS $$
DECLARE n int; s int;
BEGIN
  FOR i IN 1..40 LOOP
    SELECT c, v INTO n, s FROM dblink('dbname=logical_copy',
      'SELECT (SELECT count(*) FROM notes)::int, (SELECT last_value FROM notes_id_seq)::int') AS t(c int, v int);
    IF n = want_rows AND s = want_seq THEN RETURN true; END IF;
    PERFORM pg_sleep(0.25);
  END LOOP;
  RETURN false;
END $$;
\c logical_copy
CREATE TABLE notes (id serial PRIMARY KEY, body text);
```

```view
SELECT 'this playground' AS database, count(*) AS notes, (SELECT last_value FROM notes_id_seq) AS id_sequence FROM notes
UNION ALL
SELECT 'logical_copy', c.notes, c.seq
FROM dblink('dbname=logical_copy', 'SELECT (SELECT count(*) FROM notes), (SELECT last_value FROM notes_id_seq)') AS c(notes bigint, seq bigint);
```

```cleanup
\set ON_ERROR_STOP off
\c logical_copy
ALTER SUBSCRIPTION sub DISABLE;
ALTER SUBSCRIPTION sub SET (slot_name = NONE);
DROP SUBSCRIPTION sub;
\c postgres
SELECT pg_drop_replication_slot('sub') FROM pg_replication_slots WHERE slot_name = 'sub';
DROP DATABASE IF EXISTS logical_copy WITH (FORCE);
```

**Logical replication** copies changes from one database to another as they happen: each insert, update, and delete, row by row. It's how you keep a reporting database in sync, send some tables to another service, or move to a new server with almost no downtime.

It has two sides:

- The **publisher** has the data. A **publication** names what to share: `CREATE PUBLICATION pub FOR ALL TABLES`.
- The **subscriber** receives it. A **subscription** connects to the publisher and applies its changes: `CREATE SUBSCRIPTION sub CONNECTION '...' PUBLICATION pub`. It first copies the existing rows, then streams every new change.

On the publisher, a **replication slot** remembers how far the subscriber has read, so nothing is lost if it disconnects for a while. That cuts both ways: while a slot isn't being read, the publisher keeps the WAL it needs, and the disk fills up. Drop slots you no longer use. `pgdemo doctor` looks for leftover ones.

The tables must already exist on the subscriber. Schema changes aren't replicated, only rows.

**Sequences are new in Postgres 19.** Before 19, row changes were replicated, but sequences, the counters behind `serial` IDs, weren't. After switching over to the copy, its sequences were behind, and the next insert reused an ID that already existed. Now a publication can include them with `FOR ALL TABLES, ALL SEQUENCES`, and on the subscriber **`ALTER SUBSCRIPTION sub REFRESH SEQUENCES`** brings them up to date. Sequences sync when you ask, not with every change, so refresh just before a switchover.

**Where it came from.** Postgres 9.4 (2014) added **logical decoding**, which reads the WAL back as row changes, the same machinery pg_squeeze uses. Built on it, the `pglogical` extension from 2ndQuadrant did replication as an add-on, and that work was built into core in Postgres 10 (2017). Older tools like Slony and Londiste did it with triggers. Syncing sequences is new in Postgres 19 (2026).

Here both sides are on the same server: this playground publishes, and a second database, `logical_copy`, subscribes. Slots and subscriptions belong to the whole server, so leaving the lesson removes them.

**The Live View** compares the two databases:

| Column | Meaning |
|---|---|
| `notes` | Rows in table `notes` |
| `id_sequence` | The `notes_id_seq` counter behind the `id` column |

## Try it

Publish every table and sequence, and create the replication slot the subscription will read from. On a single server, the slot has to be created separately like this, before the subscription.

```sql
CREATE PUBLICATION pub FOR ALL TABLES, ALL SEQUENCES;
SELECT slot_name FROM pg_create_logical_replication_slot('sub', 'pgoutput');
```

Subscribe from `logical_copy`, in a shell, since it's a different database. The existing note is copied over, and the Live View shows 1 note on each side.

```sh
psql -d logical_copy -c "CREATE SUBSCRIPTION sub
  CONNECTION 'dbname=lesson_logical_replication user=postgres host=/var/run/postgresql'
  PUBLICATION pub WITH (create_slot = false, slot_name = 'sub')"
```

Add a note here. It arrives in the copy within a moment. But in the Live View, the copy's `id_sequence` stays behind, because sequences don't sync with every change.

```sql
INSERT INTO notes (body) VALUES ('second');
```

Ask the subscriber to sync its sequences. The copy's `id_sequence` catches up to 2, so if the app switched to the copy now, the next ID would be 3, not a duplicate.

```sh
psql -d logical_copy -c "ALTER SUBSCRIPTION sub REFRESH SEQUENCES"
```

## Challenge

Add a third note here, with any text, then bring the copy fully up to date: 3 notes and its sequence at 3.

Hint: Insert the note in the SQL pane, then run the last step's shell command from the SQL pane with \!.

```solution
INSERT INTO notes (body) VALUES ('third');
\! psql -d logical_copy -c "ALTER SUBSCRIPTION sub REFRESH SEQUENCES"
```

```check
SELECT copy_has(3, 3);
```
