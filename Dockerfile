FROM postgres:19
RUN apt-get update && apt-get install -y --no-install-recommends \
    postgresql-19-cron postgresql-19-partman postgresql-19-pgvector \
    postgresql-19-postgis-3 postgresql-19-postgis-3-scripts \
    postgresql-19-repack postgresql-19-squeeze pgbouncer \
 && rm -rf /var/lib/apt/lists/*
