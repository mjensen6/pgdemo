#!/usr/bin/env bash
# Runs one lesson against a fresh database called "lesson".
set -euo pipefail
file=$1
psql -X -q -d postgres -c 'DROP DATABASE IF EXISTS lesson WITH (FORCE)' -c 'CREATE DATABASE lesson' 2>/dev/null
case $file in
  *.sql) tail -n +2 "$file" | psql -X -q -a -P pager=off -v ON_ERROR_STOP=1 -d lesson -f - ;;
  *.sh)  bash "$file" ;;
esac
