---
title: Incremental backup
tldr: Back up only the blocks that changed since the last backup, then combine the pieces into one you can restore. Built into Postgres 17.
---

```setup
\! rm -rf /var/lib/postgresql/pgdemo-backups && mkdir -p /var/lib/postgresql/pgdemo-backups
CREATE TABLE t AS SELECT i, repeat('x', 100) AS pad FROM generate_series(1, 200000) i;
```

```view
SELECT d AS backup,
       (SELECT pg_size_pretty(sum((f->>'Size')::bigint))
        FROM jsonb_array_elements(pg_read_file(b.dir || d || '/backup_manifest', true)::jsonb -> 'Files') f) AS size,
       coalesce(pg_read_file(b.dir || d || '/backup_label', true) LIKE '%INCREMENTAL FROM%', false) AS incremental
FROM (SELECT '/var/lib/postgresql/pgdemo-backups/' AS dir) b, pg_ls_dir(b.dir, true, false) d
ORDER BY (pg_stat_file(b.dir || d)).modification;
```

```cleanup
\! rm -rf /var/lib/postgresql/pgdemo-backups
```

A **physical backup** is a copy of the server's data files, taken while it runs. **`pg_basebackup`** takes one. It copies every file, so a 500 GB database means 500 GB copied every night, even if only a few gigabytes changed that day.

An **incremental backup** copies only the blocks that changed since an earlier backup. You give `pg_basebackup` the earlier backup's **manifest**, the list of files and what was in them, and it sends just the differences:

```
pg_basebackup -D full -c fast
pg_basebackup -D incr -c fast --incremental=full/backup_manifest
```

You can chain them: a full backup on Sunday, then each night an incremental based on the night before.

**You can't restore an incremental backup on its own**, since it's mostly holes. **`pg_combinebackup`** merges a full backup and its chain of incrementals into one complete, ordinary backup. **`pg_verifybackup`** checks a backup against its manifest, so you know it's intact before you need it.

**How Postgres knows what changed.** With **`summarize_wal = on`**, a background process reads the write-ahead log, the record of every change, and keeps a summary of which blocks changed. This course's server has it on.

**The limits.** Restores take longer, because you combine first. Every link in a chain has to survive: lose one incremental, and the later ones are useless. Dedicated tools add scheduling, compression, and remote storage on top.

**Where it came from.** Built-in incremental backup is new in Postgres 17 (2024), written by Robert Haas. Before that, incremental backups needed outside tools such as **pgBackRest** and **Barman**, which work out the changes themselves and are still popular for everything around a backup.

This lesson shares the whole server: a backup copies every database on it, not just this playground. The backups are stored in `/var/lib/postgresql/pgdemo-backups` inside the course container, and leaving the lesson deletes them.

**The Live View** lists the backups, oldest first:

| Column | Meaning |
|---|---|
| `backup` | Its folder name |
| `size` | The data it holds, from its manifest |
| `incremental` | Whether it only holds changes since an earlier backup |

## Try it

Take a full backup of the whole server. Watch it appear in the Live View.

```sh
cd /var/lib/postgresql/pgdemo-backups
pg_basebackup -D full -c fast
echo "Full backup done."
```

Change a little data: add one row.

```sql
INSERT INTO t VALUES (999999, 'added after the full backup');
```

Take an incremental backup, based on the full one. It lands next to it in the Live View, and it's a small fraction of the size. `pg_switch_wal()` closes the current WAL file, so the summary covers the new row straight away.

```sh
cd /var/lib/postgresql/pgdemo-backups
psql -Atqc "SELECT pg_switch_wal()" >/dev/null; sleep 1
pg_basebackup -D incr -c fast --incremental=full/backup_manifest
echo "Incremental backup done."
```

Combine the two into one complete backup, then verify it. `restored` is a normal full backup you could start a server from.

```sh
cd /var/lib/postgresql/pgdemo-backups
pg_combinebackup full incr -o restored
pg_verifybackup -n restored
```

## Challenge

Extend the chain: take a second incremental backup, into the folder `incr2`, based on `incr`. Run it from the SQL pane.

Hint: Use \! and the pg_basebackup command from the incremental step. Paths need the full folder: /var/lib/postgresql/pgdemo-backups/incr2, and so on.

```solution
\! pg_basebackup -D /var/lib/postgresql/pgdemo-backups/incr2 -c fast --incremental=/var/lib/postgresql/pgdemo-backups/incr/backup_manifest
```

```check
SELECT coalesce(pg_read_file('/var/lib/postgresql/pgdemo-backups/incr2/backup_label', true) LIKE '%INCREMENTAL FROM%', false);
```
