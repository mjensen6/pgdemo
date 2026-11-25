---
title: Row-level security
tldr: Two users query the same table and see different rows, because the database itself filters them, whatever query the app sends.
---

```setup
DO $$ BEGIN CREATE ROLE rls_alice NOLOGIN; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE ROLE rls_bob NOLOGIN; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
CREATE TABLE documents (id int PRIMARY KEY, owner text NOT NULL, title text, public bool NOT NULL DEFAULT false);
INSERT INTO documents VALUES
  (1, 'rls_alice', 'Alice: launch plan', false),
  (2, 'rls_alice', 'Alice: meeting notes', true),
  (3, 'rls_bob',   'Bob: budget', false),
  (4, 'rls_bob',   'Bob: ideas', false);
GRANT SELECT, INSERT ON documents TO rls_alice, rls_bob;
-- Shows what a user sees, by briefly becoming them. Used by the Live View and the check.
CREATE FUNCTION visible_to(who text) RETURNS text[] LANGUAGE plpgsql AS $$
DECLARE titles text[];
BEGIN
  EXECUTE format('SET LOCAL ROLE %I', who);
  SELECT array_agg(title ORDER BY id) INTO titles FROM documents;
  RESET ROLE;
  RETURN coalesce(titles, '{}');
END $$;
```

```view
SELECT u AS "user", array_to_string(visible_to(u), ', ') AS sees
FROM unnest(ARRAY['rls_alice', 'rls_bob']) u
UNION ALL
SELECT 'policies', coalesce(string_agg(policyname || ' (' || cmd || ')', ', '), '(none)')
FROM pg_policies WHERE tablename = 'documents';
```

Most apps keep everyone's data in the same tables and filter by user in the app: `WHERE owner = $current_user`. Forget that `WHERE` in one query, one report, or one new endpoint, and one customer sees another's data.

**Row-level security** (RLS) moves the rule into the database. You turn it on for a table and write **policies**, and from then on Postgres adds the filter to **every** query on that table, whoever wrote it:

```sql
ALTER TABLE documents ENABLE ROW LEVEL SECURITY;
CREATE POLICY own ON documents USING (owner = current_user);
```

- **`USING (...)`** decides which existing rows a user can see, update, or delete. Rows that fail it are simply invisible, as if they weren't there.
- **`WITH CHECK (...)`** decides which new or changed rows a user may write. Without it, `USING` is used for this too, so Bob can't insert a document owned by Alice.
- **`FOR SELECT`**, `FOR INSERT`, and so on limit a policy to one kind of command.
- A user may do something if **any** policy allows it. Policies add access, they don't take it away.

`current_user` is the role running the query. Apps that connect as one role often set the user in a setting instead and use `current_setting('app.user_id')` in the policy.

**Who skips the rules.** Superusers, and the table's owner, bypass RLS. The course's own sessions, the SQL pane included, connect as a superuser, so to see the policies at work you **become** a normal role with `SET ROLE rls_bob`, and go back with `RESET ROLE`. Once RLS is on, a role with no policy that applies sees no rows at all.

**Where it came from.** Row-level security arrived in Postgres 9.5 (2016). Before that, people approximated it with views, including `security_barrier` views (9.2, 2012). It's now the foundation of services like Supabase, where the browser queries the database through an API and RLS policies decide what each user can reach.

Two roles exist for this lesson, `rls_alice` and `rls_bob`. Roles belong to the whole server, so they're named after the lesson.

**The Live View** shows the titles each user can see, by briefly becoming them, and the table's policies.

## Try it

As a superuser, you see everything. Both users can see everything too: they're allowed to `SELECT` from the table, and nothing filters the rows yet.

```sql
SELECT id, owner, title, public FROM documents ORDER BY id;
```

Turn on RLS and add a policy: you see your own documents. In the Live View, Alice and Bob now each see only their own.

```sql
ALTER TABLE documents ENABLE ROW LEVEL SECURITY;
CREATE POLICY own ON documents USING (owner = current_user);
```

Become Bob. `SELECT *` returns only his rows. Then he tries to slip in a document owned by Alice, and the policy refuses it. `RESET ROLE` makes you the superuser again.

```sql
SET ROLE rls_bob;
SELECT id, owner, title FROM documents;
INSERT INTO documents VALUES (9, 'rls_alice', 'Forged by Bob', false);
RESET ROLE;
```

## Challenge

Documents marked `public` should be readable by everyone, while private ones stay with their owner, and nobody gains the right to change what isn't theirs. Add a policy for that. Bob should then also see "Alice: meeting notes".

Hint: CREATE POLICY like the second step, with a new name, FOR SELECT, and USING (public).

```solution
CREATE POLICY public_read ON documents FOR SELECT USING (public);
```

```check
SELECT visible_to('rls_bob') = '{"Alice: meeting notes","Bob: budget","Bob: ideas"}'
   AND visible_to('rls_alice') = '{"Alice: launch plan","Alice: meeting notes"}'
   AND NOT EXISTS (SELECT FROM pg_policies WHERE tablename = 'documents' AND cmd IN ('ALL', 'UPDATE', 'DELETE', 'INSERT')
                   AND policyname <> 'own');
```
