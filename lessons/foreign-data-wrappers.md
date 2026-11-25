---
title: Foreign data wrappers
tldr: Query a table in another database as if it were local, join it with your own tables, and let Postgres send the filtering to the other side.
---

```setup
DROP DATABASE IF EXISTS fdw_warehouse WITH (FORCE);
CREATE DATABASE fdw_warehouse;
CREATE EXTENSION postgres_fdw;
CREATE TABLE orders (id int PRIMARY KEY, product_id int, qty int);
INSERT INTO orders VALUES (1, 1, 2), (2, 3, 1), (3, 1, 5), (4, 2, 1);
\c fdw_warehouse
CREATE TABLE products (id int PRIMARY KEY, name text, price numeric);
INSERT INTO products VALUES (1, 'Lamp', 30), (2, 'Tent', 120), (3, 'Stove', 45);
CREATE TABLE inventory (product_id int PRIMARY KEY, in_stock int);
INSERT INTO inventory VALUES (1, 12), (2, 0), (3, 4);
```

```view
SELECT c.relname AS foreign_table, s.srvname AS server, array_to_string(ft.ftoptions, ', ') AS remote
FROM pg_foreign_table ft JOIN pg_class c ON c.oid = ft.ftrelid JOIN pg_foreign_server s ON s.oid = ft.ftserver
UNION ALL
SELECT '(none yet)', coalesce((SELECT string_agg(srvname, ', ') FROM pg_foreign_server), '(no servers)'), ''
WHERE NOT EXISTS (SELECT FROM pg_foreign_table)
ORDER BY 1;
```

Data rarely lives in one place. The orders are here, but the product catalog is in the warehouse team's database. The usual answer is an export job that copies data across every night, and it's always a little out of date.

A **foreign data wrapper** (FDW) lets Postgres query data that lives somewhere else, live, as if it were a local table. **`postgres_fdw`** connects to other Postgres databases. Setting it up takes three objects:

1. A **server**: where the other database is. `CREATE SERVER warehouse FOREIGN DATA WRAPPER postgres_fdw OPTIONS (dbname 'fdw_warehouse')`. For another machine you'd add `host` and `port`.
2. A **user mapping**: which user to log in as there. `CREATE USER MAPPING FOR CURRENT_USER SERVER warehouse OPTIONS (user 'postgres')`.
3. **Foreign tables**: local names for remote tables. `IMPORT FOREIGN SCHEMA` creates them for you, all at once or `LIMIT TO (some, tables)`.

After that, `SELECT`, `JOIN`, and even `INSERT` and `UPDATE` work on the foreign tables.

**Pushdown.** postgres_fdw sends work to the remote side where it can: `WHERE` filters, sorting, aggregates, and joins between tables on the same server. So filtering a million remote rows for one product sends one row back, not a million. `EXPLAIN (VERBOSE)` shows the **Remote SQL** it sends.

**The costs.** Every query reaches across the network, so a foreign table is slower than a local one, and the remote database being down means your query fails. A join between a local and a remote table may pull a lot of rows across. Use it for live lookups and occasional reports. Copy the data, or use logical replication (lesson 40), when you query it constantly.

**Wrappers for other things.** There are FDWs for MySQL, Oracle, SQL Server, SQLite, MongoDB, CSV files (`file_fdw`, which ships with Postgres), and more, mostly third-party.

**Where it came from.** FDWs follow SQL/MED, the "management of external data" part of the SQL standard. Postgres 9.1 (2011) added the framework, read-only. Postgres 9.3 (2013) added `postgres_fdw` and writes through foreign tables, and 9.5 (2016) added `IMPORT FOREIGN SCHEMA`. Before that, the `dblink` contrib module could run queries in another database, but as function calls rather than tables.

Here the other database, `fdw_warehouse`, is on the same server, so no network is involved, but it works the same way.

**The Live View** lists the foreign tables in this playground, and which server each points at.

## Try it

Describe the warehouse: a server, and a user mapping for logging in to it. The Live View shows the server, with no foreign tables yet.

```sql
CREATE SERVER warehouse FOREIGN DATA WRAPPER postgres_fdw OPTIONS (dbname 'fdw_warehouse');
CREATE USER MAPPING FOR CURRENT_USER SERVER warehouse OPTIONS (user 'postgres');
```

Import the warehouse's `products` table. It appears in the Live View as a foreign table, and you query it like any other.

```sql
IMPORT FOREIGN SCHEMA public LIMIT TO (products) FROM SERVER warehouse INTO public;
SELECT * FROM products;
```

Join your local orders with the remote products, to get each order's value.

```sql
SELECT o.id, p.name, o.qty, o.qty * p.price AS value
FROM orders o JOIN products p ON p.id = o.product_id ORDER BY o.id;
```

See the pushdown. The plan's `Remote SQL` line shows the query Postgres sends to the warehouse, with the `WHERE` included, so only matching rows come back.

```sql
EXPLAIN (VERBOSE, COSTS OFF) SELECT name FROM products WHERE price > 40;
```

## Challenge

The warehouse also has an `inventory` table (product_id, in_stock). Make it available here as a foreign table, so you can check stock before taking an order.

Hint: The import step, limited to the other table.

```solution
IMPORT FOREIGN SCHEMA public LIMIT TO (inventory) FROM SERVER warehouse INTO public;
```

```check
SELECT EXISTS (SELECT FROM pg_foreign_table ft JOIN pg_class c ON c.oid = ft.ftrelid WHERE c.relname = 'inventory');
```
