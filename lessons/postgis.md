---
title: PostGIS
tldr: Store places as real geography and ask spatial questions in SQL, such as how far apart two places are and what's within 300 miles.
---

```setup
CREATE EXTENSION postgis;
CREATE TABLE cities (name text PRIMARY KEY, geo geography(point), region text);
INSERT INTO cities (name, geo) VALUES
  ('Chicago',     'POINT(-87.63 41.88)'),
  ('Milwaukee',   'POINT(-87.91 43.04)'),
  ('Detroit',     'POINT(-83.05 42.33)'),
  ('St. Louis',   'POINT(-90.20 38.63)'),
  ('Minneapolis', 'POINT(-93.27 44.98)'),
  ('New York',    'POINT(-74.01 40.71)'),
  ('Denver',      'POINT(-104.99 39.74)');
```

```view
SELECT name, ST_AsText(geo) AS location,
       round((ST_Distance(geo, 'POINT(-87.63 41.88)') / 1609.34)::numeric) AS miles_from_chicago, region
FROM cities ORDER BY miles_from_chicago;
```

**PostGIS** is an extension that teaches Postgres about space: points, lines, and shapes on a map, and the questions you ask about them. How far is it from here to there? Which stores are within 5 miles? Which delivery zone is this address in? Do these two areas overlap?

It adds two column types:

- **`geometry`**: shapes on a flat plane. Fast, and right for small areas or projected maps.
- **`geography`**: points on the real, round Earth, with longitude and latitude in degrees. Distances come out in meters. It's what this lesson uses.

A point is written as text in the standard **WKT** format, longitude first: `'POINT(-87.63 41.88)'` is Chicago.

A few functions do most of the work:

| Function | Meaning |
|---|---|
| `ST_Distance(a, b)` | Distance between two places. Meters for `geography` |
| `ST_DWithin(a, b, d)` | True if they're within `d` meters of each other. It can use an index |
| `ST_AsText(g)` | Show a value as readable WKT |

A mile is 1,609.34 meters, so `300 * 1609.34` means 300 miles.

**Indexes.** A **GiST** index on the column makes `ST_DWithin` fast: Postgres skips everything outside the search area instead of measuring the distance to every row.

**Where it came from.** PostGIS is a third-party extension, first released in 2001 by Refractions Research, a Canadian consulting firm. Version 1.0 came in 2005. It follows the Open Geospatial Consortium's standards, which is where the `ST_` names come from. It's one of the most widely used open-source GIS tools and a big reason people choose Postgres for map data. It has never been part of core Postgres, but it's offered almost everywhere.

**The Live View** shows table `cities`, nearest to Chicago first:

| Column | Meaning |
|---|---|
| `location` | Longitude and latitude, as WKT |
| `miles_from_chicago` | `ST_Distance` to Chicago, converted to miles |
| `region` | A label you'll set in the challenge |

## Try it

Distance from Chicago to every other city, in miles. `ST_Distance` gives meters, so divide by 1,609.34.

```sql
SELECT b.name, round((ST_Distance(a.geo, b.geo) / 1609.34)::numeric) AS miles
FROM cities a, cities b
WHERE a.name = 'Chicago' AND b.name <> 'Chicago' ORDER BY miles;
```

```expect
Milwaukee   |    81
Detroit     |   237
St. Louis   |   262
Minneapolis |   355
New York    |   713
Denver      |   920
```

Every city within 300 miles of Chicago, Chicago included. `ST_DWithin` takes the distance in meters.

```sql
SELECT name FROM cities WHERE ST_DWithin(geo, 'POINT(-87.63 41.88)', 300 * 1609.34);
```

```expect
Chicago
Milwaukee
Detroit
St. Louis
(4 rows)
```

Add a GiST index, so searches like that skip far-away rows instead of measuring each one.

```sql
CREATE INDEX cities_geo ON cities USING gist (geo);
```

```expect
CREATE INDEX
```

## Challenge

Set `region` to `'midwest'` for every city within 400 miles of Chicago, Chicago included, and leave the others alone.

Hint: UPDATE cities SET region = ... WHERE, with the ST_DWithin test from the second step and a bigger distance.

```solution
UPDATE cities SET region = 'midwest' WHERE ST_DWithin(geo, 'POINT(-87.63 41.88)', 400 * 1609.34);
```

```check
SELECT array_agg(name ORDER BY name) = '{Chicago,Detroit,Milwaukee,Minneapolis,"St. Louis"}'
FROM cities WHERE region = 'midwest';
```
