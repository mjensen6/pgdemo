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

# The River lesson's demo app. River needs a newer Go than the player.
FROM golang:1.26-trixie AS river
WORKDIR /src
COPY river-demo/go.mod river-demo/go.sum ./
RUN go mod download
COPY river-demo/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /river-demo .

# PostgREST for its lesson, a pinned release checked against its SHA-256.
# Only x86-64 has a static build. ARM uses the Ubuntu build, which links
# against the libraries installed in the final image.
FROM debian:trixie-slim AS postgrest
ARG TARGETARCH
ARG POSTGREST_VERSION=v13.0.8
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl xz-utils \
 && case "$TARGETARCH" in \
      amd64) asset=linux-static-x86-64; sum=eb4471a7b8112a54e103539f2348c4d7f1c5b9b390649700ad62836e9211e279 ;; \
      arm64) asset=ubuntu-aarch64;      sum=3d3f09c7d0bed34fb9e1158f7c4a91c23f2fce01c7c1a6f7275214f58d829d20 ;; \
      *) echo "No PostgREST build for $TARGETARCH" >&2; exit 1 ;; \
    esac \
 && curl -fsSLo /tmp/postgrest.tar.xz "https://github.com/PostgREST/postgrest/releases/download/${POSTGREST_VERSION}/postgrest-${POSTGREST_VERSION}-${asset}.tar.xz" \
 && echo "$sum  /tmp/postgrest.tar.xz" | sha256sum -c - \
 && tar -xJf /tmp/postgrest.tar.xz -C /usr/local/bin postgrest

FROM postgres:19
RUN apt-get update && apt-get install -y --no-install-recommends \
    tmux ncurses-term pgcli pspg pg-activity curl libgmp10 \
    postgresql-19-cron postgresql-19-partman postgresql-19-pgvector \
    postgresql-19-postgis-3 postgresql-19-postgis-3-scripts \
    postgresql-19-repack postgresql-19-squeeze postgresql-19-hypopg \
    postgresql-19-pg-ivm postgresql-19-pgrouting postgresql-19-hll \
    postgresql-19-pgaudit postgresql-19-pgtap pgbouncer \
 && rm -rf /var/lib/apt/lists/*
COPY --from=player /pgdemo-player /usr/local/bin/pgdemo-player
COPY --from=lazysql /out/lazysql /usr/local/bin/lazysql
COPY --from=river /river-demo /usr/local/bin/river-demo
COPY --from=postgrest /usr/local/bin/postgrest /usr/local/bin/postgrest
COPY lessons /opt/pgdemo/lessons
ENV LANG=C.UTF-8 DO_NOT_TRACK=1 LAZYSQL_NO_UPDATE_CHECK=1
