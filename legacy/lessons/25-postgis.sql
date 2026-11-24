-- 25 PostGIS: maps and distance in SQL
CREATE EXTENSION postgis;
CREATE TABLE cities (name text, geo geography(point));
INSERT INTO cities VALUES
  ('Chicago', 'POINT(-87.63 41.88)'), ('Milwaukee', 'POINT(-87.91 43.04)'),
  ('St. Louis', 'POINT(-90.20 38.63)'), ('Denver', 'POINT(-104.99 39.74)');
CREATE INDEX ON cities USING gist (geo);
-- Distance from Chicago, in miles.
SELECT b.name, round((ST_Distance(a.geo, b.geo) / 1609.34)::numeric) AS miles
FROM cities a, cities b WHERE a.name = 'Chicago' AND b.name <> 'Chicago' ORDER BY miles;
-- Everything within 300 miles.
SELECT name FROM cities WHERE ST_DWithin(geo, 'POINT(-87.63 41.88)', 300 * 1609.34);
