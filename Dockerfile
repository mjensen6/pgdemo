# The whole course runs in here: Postgres 19, the extensions the lessons
# use, the terminal tools, and the pgdemo player. Nothing is installed on
# the host.

FROM golang:1.25-trixie AS player
WORKDIR /src
COPY player/go.mod player/go.sum ./
RUN go mod download
COPY player/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /pgdemo-player .

# lazysql powers the Browse tab. Pinned, and built from source, so it has
# no telemetry endpoint compiled in.
FROM golang:1.25-trixie AS lazysql
ARG LAZYSQL_VERSION=v0.7.0
RUN CGO_ENABLED=0 GOBIN=/out go install -trimpath \
    -ldflags="-s -w -X main.version=${LAZYSQL_VERSION}" \
    github.com/jorgerojas26/lazysql@${LAZYSQL_VERSION}

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
COPY --from=lazysql /out/lazysql /usr/local/bin/lazysql
COPY lessons /opt/pgdemo/lessons
ENV LANG=C.UTF-8 DO_NOT_TRACK=1 LAZYSQL_NO_UPDATE_CHECK=1
