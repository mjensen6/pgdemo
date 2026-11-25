---
title: Recursive queries
tldr: WITH RECURSIVE walks trees and graphs, such as an org chart, a folder tree, or a comment thread, in a single query.
---

```setup
CREATE TABLE employees (id int PRIMARY KEY, name text, manager_id int REFERENCES employees, team text);
INSERT INTO employees (id, name, manager_id) VALUES
  (1, 'Ada', NULL),
  (2, 'Grace', 1), (3, 'Linus', 1),
  (4, 'Ken', 2), (5, 'Barbara', 2), (6, 'Guido', 3),
  (7, 'Margaret', 4), (8, 'Dennis', 4), (9, 'Bjarne', 6),
  (10, 'Frances', 7);
```

```view
SELECT e.id, e.name, m.name AS manager, e.team
FROM employees e LEFT JOIN employees m ON m.id = e.manager_id ORDER BY e.id;
```

Lots of data is a **tree**: an org chart where each person has a manager, folders inside folders, replies to replies. Each row points at its parent. One join finds someone's manager. But "everyone under Grace, at every level" needs as many joins as there are levels, and you don't know how many that is.

**`WITH RECURSIVE`** handles any depth. It has two parts joined by `UNION ALL`:

```sql
WITH RECURSIVE team AS (
  SELECT id, name FROM employees WHERE name = 'Grace'           -- 1. the starting rows
  UNION ALL
  SELECT e.id, e.name FROM employees e JOIN team t ON e.manager_id = t.id   -- 2. one more level
)
SELECT * FROM team;
```

1. The **anchor** runs once and gives the starting rows.
2. The **recursive part** refers to the query's own name, `team`. Postgres runs it on the rows found in the previous round, adds what it finds, and repeats until a round finds nothing new.

You can carry values along: a `depth` that goes up by 1 each round, or a `path` that grows. Recursion can also walk upward, from a person to the top, by joining on the parent instead of the child.

**The danger is a cycle.** If the data loops, say A manages B and B manages A, the query never stops finding "new" rows. Add a depth limit like `WHERE depth < 20`, or the `CYCLE` clause, which detects loops for you.

**Where it came from.** Recursive queries are in the SQL standard since SQL:1999. Postgres got them in 8.4 (2009). Before that, the `tablefunc` contrib module offered a `connectby()` function for walking trees, and Oracle had its own `CONNECT BY` syntax. Postgres 14 (2021) added the standard `SEARCH` and `CYCLE` clauses.

**The Live View** shows the org chart:

| Column | Meaning |
|---|---|
| `manager` | Who they report to. Ada reports to nobody |
| `team` | A label you'll set in the challenge |

## Try it

The simplest recursive query: start at 1, add 1 each round, stop at 5.

```sql
WITH RECURSIVE n AS (
  SELECT 1 AS i
  UNION ALL
  SELECT i + 1 FROM n WHERE i < 5
)
SELECT i FROM n;
```

Everyone under Ada, at every level, with how far down they are. Each round finds the people whose manager was found in the round before.

```sql
WITH RECURSIVE org AS (
  SELECT id, name, 0 AS depth FROM employees WHERE manager_id IS NULL
  UNION ALL
  SELECT e.id, e.name, o.depth + 1 FROM employees e JOIN org o ON e.manager_id = o.id
)
SELECT repeat('  ', depth) || name AS chart, depth FROM org ORDER BY depth, name;
```

Walk upward instead: from Frances to the top, building the chain of managers as a path. This time each round joins on the manager.

```sql
WITH RECURSIVE up AS (
  SELECT id, manager_id, name::text AS path FROM employees WHERE name = 'Frances'
  UNION ALL
  SELECT e.id, e.manager_id, u.path || ' → ' || e.name FROM employees e JOIN up u ON e.id = u.manager_id
)
SELECT path FROM up WHERE manager_id IS NULL;
```

## Challenge

Set `team` to `'platform'` for Grace and everyone under her, at every level, and nobody else.

Hint: UPDATE employees SET team = 'platform' WHERE id IN (...), with the intro's recursive query inside the parentheses, selecting only id.

```solution
UPDATE employees SET team = 'platform' WHERE id IN (
  WITH RECURSIVE team AS (
    SELECT id FROM employees WHERE name = 'Grace'
    UNION ALL
    SELECT e.id FROM employees e JOIN team t ON e.manager_id = t.id
  )
  SELECT id FROM team
);
```

```check
-- Grace, Ken, Barbara, Margaret, Dennis, Frances.
SELECT array_agg(id ORDER BY id) = '{2,4,5,7,8,10}' FROM employees WHERE team = 'platform';
```
