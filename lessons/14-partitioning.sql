-- 14 Partitioning: split one big table into pieces
CREATE TABLE events (at date, msg text) PARTITION BY RANGE (at);
CREATE TABLE events_2025 PARTITION OF events FOR VALUES FROM ('2025-01-01') TO ('2026-01-01');
CREATE TABLE events_2026 PARTITION OF events FOR VALUES FROM ('2026-01-01') TO ('2027-01-01');
INSERT INTO events VALUES ('2025-06-01', 'old'), ('2026-06-01', 'new');
SELECT tableoid::regclass AS stored_in, * FROM events;
-- Postgres reads only the partition it needs.
EXPLAIN (COSTS OFF) SELECT * FROM events WHERE at = '2026-06-01';
-- Deleting a year of data is instant.
DROP TABLE events_2025;
SELECT * FROM events;
