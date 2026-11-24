# 22 Incremental backup (PG17): back up only what changed
. "$(dirname "$0")/../lib/helpers.sh"
d=$(mktemp -d); trap 'rm -rf "$d"' EXIT
quiet "CREATE TABLE t AS SELECT i, repeat('x', 100) AS pad FROM generate_series(1, 200000) i"
say "Full backup."
echo 'pg_basebackup -D full -c fast'
pg_basebackup -D "$d/full" -c fast
quiet "INSERT INTO t VALUES (999999, 'added after the full backup')"
quiet "SELECT pg_switch_wal()"; sleep 1.5
say "Incremental backup, based on the full one."
echo 'pg_basebackup -D incr -c fast --incremental=full/backup_manifest'
pg_basebackup -D "$d/incr" -c fast --incremental="$d/full/backup_manifest"
du -sh "$d/full" "$d/incr" | sed "s#$d/##"
say "Combine them into one restorable backup, then verify it."
echo 'pg_combinebackup full incr -o restored && pg_verifybackup restored'
pg_combinebackup "$d/full" "$d/incr" -o "$d/restored"
pg_verifybackup -n "$d/restored"
