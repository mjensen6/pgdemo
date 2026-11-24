#!/usr/bin/env bash
# Course test: plays every built lesson from a clean start. Each lesson's
# check must fail before its solution and pass after it. Pass -v to print
# every step's real output, which is worth reading: no error does not mean
# a lesson works. It rebuilds each lesson's playground, so your own work
# in those lessons is reset.
set -euo pipefail
cd "$(dirname "$0")/.."
docker compose up -d --build --wait >/dev/null
docker compose exec -T -u postgres db pgdemo-player test "$@"
