---
title: pgTAP
tldr: Unit tests written in SQL, for your functions, tables, and constraints, run inside the database and rolled back afterwards.
---

```setup
CREATE EXTENSION pgtap;
CREATE TABLE products (id int PRIMARY KEY, name text NOT NULL, price numeric NOT NULL CHECK (price >= 0));
CREATE FUNCTION discount(price numeric, pct int) RETURNS numeric LANGUAGE sql
  AS $$ SELECT round(price * (100 - pct) / 100, 2) $$;
CREATE FUNCTION shipping(weight_kg numeric) RETURNS numeric LANGUAGE sql
  AS $$ SELECT CASE WHEN weight_kg <= 1 THEN 5 ELSE 5 + ceil(weight_kg - 1) * 2 END $$;
```

```view
SELECT p.proname || '()' AS test_function,
       CASE WHEN r IS NULL THEN '' ELSE r END AS last_line_of_results
FROM pg_proc p
LEFT JOIN LATERAL (SELECT x AS r FROM runtests('^' || p.proname || '$') x OFFSET 0) t ON t.r ~ '^(not )?ok'
WHERE p.proname LIKE 'test%' AND p.pronamespace = 'public'::regnamespace
UNION ALL
SELECT '(none yet)', '' WHERE NOT EXISTS (SELECT FROM pg_proc WHERE proname LIKE 'test%' AND pronamespace = 'public'::regnamespace);
```

Database code needs tests too: functions with logic in them, constraints that must reject bad data, a schema your app expects. You can test them from the app, but it's slow and indirect.

**pgTAP** is an extension for writing tests in SQL, run inside the database. Each test is a function call that checks one thing:

| Function | Passes when |
|---|---|
| `ok(condition, 'description')` | The condition is true |
| `is(got, expected, 'description')` | The two values are equal |
| `has_table('name')`, `has_column('table', 'column')` | It exists |
| `throws_ok('SQL', 'errcode')` | Running the SQL fails with that error |

A test run starts with `plan(n)`, the number of tests you expect, and ends with `finish()`. The results come out in **TAP**, the Test Anything Protocol: `ok 1 - description` or `not ok 2 - description`, with details when something fails. Run tests inside a transaction and `ROLLBACK` at the end, and anything a test created or changed disappears.

**Test functions.** For a suite you keep, put assertions in functions whose names start with `test`, each returning `SETOF text`, and run them all with **`runtests()`**. It runs each function in its own transaction, rolls it back, and reports.

```sql
CREATE FUNCTION test_discount() RETURNS SETOF text LANGUAGE plpgsql AS $$
BEGIN
  RETURN NEXT is(discount(100, 10), 90.00, 'ten percent off');
  RETURN NEXT is(discount(100, 0), 100.00, 'no discount');
END $$;
SELECT * FROM runtests('^test_discount$');
```

In CI, the `pg_prove` command runs test files or functions and fails the build if anything fails.

**Where it came from.** pgTAP is a third-party extension, created by David E. Wheeler in 2008. TAP itself comes from Perl's test tools of the late 1980s, which is why pgTAP's output looks like any other TAP test suite.

**The Live View** lists the test functions in this playground with their last result line from `runtests()`: `ok` when every assertion passes.

## Try it

A quick test run: a plan of 3, three checks, and a finish. All three print `ok`. The `ROLLBACK` means nothing the tests did is kept.

```sql
BEGIN;
SELECT plan(3);
SELECT has_table('products');
SELECT is(discount(100, 10), 90.00, 'ten percent off 100 is 90');
SELECT throws_ok('INSERT INTO products VALUES (1, ''Lamp'', -5)', '23514');
SELECT * FROM finish();
ROLLBACK;
```

A failing test, to see what that looks like. `not ok` comes with what it got and what it expected.

```sql
BEGIN;
SELECT plan(1);
SELECT is(discount(80, 25), 50.00, 'a quarter off 80');
SELECT * FROM finish();
ROLLBACK;
```

Keep a suite: write the tests in a function and run it with `runtests()`. It appears in the Live View with its result.

```sql
CREATE FUNCTION test_discount() RETURNS SETOF text LANGUAGE plpgsql AS $$
BEGIN
  RETURN NEXT is(discount(100, 10), 90.00, 'ten percent off');
  RETURN NEXT is(discount(100, 0), 100.00, 'no discount');
END $$;
SELECT * FROM runtests('^test_discount$');
```

## Challenge

Write a test function `test_shipping()` with at least two checks on `shipping()` that pass. Shipping costs 5 up to 1 kg, plus 2 for each started kilogram after that, so 1 kg costs 5 and 2.5 kg costs 9.

Hint: Copy test_discount from the last step, rename it, and use is() with shipping(1) and shipping(2.5).

```solution
CREATE FUNCTION test_shipping() RETURNS SETOF text LANGUAGE plpgsql AS $$
BEGIN
  RETURN NEXT is(shipping(1), 5::numeric, 'one kilogram costs 5');
  RETURN NEXT is(shipping(2.5), 9::numeric, 'two and a half kilograms cost 9');
END $$;
```

```check
-- The suite ran, has at least two assertions, and none failed.
SELECT bool_or(r ~ '^ok \d+ - public\.test_shipping')
   AND count(*) FILTER (WHERE r ~ '^\s+ok ') >= 2
   AND NOT bool_or(r ~ 'not ok')
FROM runtests('^test_shipping$') r;
```
