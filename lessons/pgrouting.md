---
title: pgRouting
tldr: Find the shortest route through a road network in SQL, and see it change when a road closes.
---

```setup
CREATE EXTENSION postgis;
CREATE EXTENSION pgrouting;
CREATE TABLE towns (id int PRIMARY KEY, name text);
INSERT INTO towns VALUES (1, 'Home'), (2, 'Mill'), (3, 'Bridge'), (4, 'Market'), (5, 'Station'), (6, 'Airport');
-- cost and reverse_cost are minutes in each direction. A negative cost means closed.
CREATE TABLE roads (id int PRIMARY KEY, source int, target int, cost float8, reverse_cost float8);
INSERT INTO roads VALUES
  (1, 1, 2, 10, 10), (2, 2, 3, 5, 5), (3, 3, 6, 10, 10), (4, 1, 4, 7, 7),
  (5, 4, 5, 8, 8), (6, 5, 6, 12, 12), (7, 2, 4, 4, 4), (8, 3, 5, 6, 6);
CREATE TABLE route (seq int, node bigint, edge bigint, cost float8, agg_cost float8);
```

```view
SELECT r.id AS road, a.name || ' – ' || b.name AS between, r.cost AS minutes,
       CASE WHEN r.cost < 0 THEN 'closed' ELSE 'open' END AS status
FROM roads r JOIN towns a ON a.id = r.source JOIN towns b ON b.id = r.target ORDER BY r.id;
```

Many questions are about **networks**: the fastest way to drive from here to the airport, which deliveries a van can reach in 15 minutes, how to route around a closed road. Graph databases and routing engines answer these, but if your map data is already in Postgres, it can answer them itself.

**pgRouting** is an extension that adds graph algorithms to Postgres. A network is an ordinary table of **edges**, roads here, with:

| Column | Meaning |
|---|---|
| `id` | The edge |
| `source`, `target` | The two **nodes**, towns here, that it connects |
| `cost` | What it costs to travel from source to target: minutes, meters, or anything |
| `reverse_cost` | The cost the other way. A negative cost means you can't go that way |

pgRouting's functions take a **query** that returns those columns, so you can filter or adjust the network on the fly, plus where to start and end. The best known is **`pgr_dijkstra`**, which finds the cheapest path:

```sql
SELECT * FROM pgr_dijkstra('SELECT id, source, target, cost, reverse_cost FROM roads', 1, 6);
```

It returns the path one step per row: `seq` (step number), `node` (town reached), `edge` (road taken from it, `-1` at the end), `cost` (that road's cost), and `agg_cost` (total so far).

`pgr_drivingDistance` finds everything reachable within a cost, and there are functions for many other problems, such as the travelling salesperson.

**The limits.** pgRouting needs a clean network, where roads that meet share a node. With real map data, such as OpenStreetMap, tools like `osm2pgrouting` build that for you. For routing a whole country in milliseconds, dedicated routing engines are faster.

**Where it came from.** pgRouting is a third-party extension, built on PostGIS (lesson 36). It began as pgDijkstra, written by Sylvain Pasche at the Swiss company Camptocamp. Orkney extended it and renamed it pgRouting, and it's now a project of OSGeo, the open-source geospatial foundation.

**The Live View** shows the road network between six towns, with each road's travel time and whether it's open.

## Try it

The fastest route from Home (1) to the Airport (6), with town names. It goes through Mill and Bridge, 25 minutes in all.

```sql
SELECT d.seq, t.name AS town, d.agg_cost AS minutes_so_far
FROM pgr_dijkstra('SELECT id, source, target, cost, reverse_cost FROM roads', 1, 6) d
JOIN towns t ON t.id = d.node ORDER BY d.seq;
```

The road between Mill and Bridge closes. A negative cost removes it from the network, and the Live View shows it closed.

```sql
UPDATE roads SET cost = -1, reverse_cost = -1 WHERE id = 2;
```

Ask again. pgRouting routes around the closure, through Market and Station: 27 minutes.

```sql
SELECT d.seq, t.name AS town, d.agg_cost AS minutes_so_far
FROM pgr_dijkstra('SELECT id, source, target, cost, reverse_cost FROM roads', 1, 6) d
JOIN towns t ON t.id = d.node ORDER BY d.seq;
```

Everywhere reachable from Home within 15 minutes, with the road still closed.

```sql
SELECT t.name AS town, d.agg_cost AS minutes
FROM pgr_drivingDistance('SELECT id, source, target, cost, reverse_cost FROM roads', 1, 15) d
JOIN towns t ON t.id = d.node ORDER BY d.agg_cost;
```

## Challenge

With the Mill–Bridge road still closed, find the fastest route from the Station (5) to Mill (2), and save it into table `route` (seq, node, edge, cost, agg_cost).

Hint: INSERT INTO route SELECT seq, node, edge, cost, agg_cost FROM pgr_dijkstra(...), with the new start and end.

```solution
INSERT INTO route
SELECT seq, node, edge, cost, agg_cost FROM pgr_dijkstra('SELECT id, source, target, cost, reverse_cost FROM roads', 5, 2);
```

```check
-- Station → Market → Mill, 12 minutes.
SELECT array_agg(node ORDER BY seq) = '{5,4,2}' AND max(agg_cost) = 12 FROM route;
```
