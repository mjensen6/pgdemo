# Shared helpers for the shell-based lessons.
set -euo pipefail
say() { printf '\n-- %s\n' "$*"; }
sql() { printf '%s\n' "$1"; psql -X -q -P pager=off -v ON_ERROR_STOP=1 -d "${2:-lesson}" -c "$1"; }
quiet() { psql -X -q -At -v ON_ERROR_STOP=1 -d "${2:-lesson}" -c "$1"; }
