# The whole course runs in here: Postgres 19, the extensions the lessons
# use, the terminal tools, and the pgdemo player. Nothing is installed on
# the host.

FROM golang:1.25-trixie AS player
WORKDIR /src
COPY player/go.mod player/go.sum ./
RUN go mod download
COPY player/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /pgdemo-player .

FROM postgres:19
RUN apt-get update && apt-get install -y --no-install-recommends \
    tmux ncurses-term pgcli pspg pg-activity \
    postgresql-19-cron postgresql-19-partman postgresql-19-pgvector \
    postgresql-19-postgis-3 postgresql-19-postgis-3-scripts \
    postgresql-19-repack postgresql-19-squeeze postgresql-19-hypopg \
    postgresql-19-pg-ivm postgresql-19-pgrouting postgresql-19-hll \
    postgresql-19-pgaudit postgresql-19-pgtap pgbouncer \
 && rm -rf /var/lib/apt/lists/*
COPY --from=player /pgdemo-player /usr/local/bin/pgdemo-player
COPY lessons /opt/pgdemo/lessons
ENV LANG=C.UTF-8
