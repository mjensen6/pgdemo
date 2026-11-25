---
title: pgaudit
tldr: A detailed log of who read or changed what, for the audits that security and compliance rules demand.
---

```setup
DO $$ BEGIN CREATE ROLE pgaudit_auditor NOLOGIN; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
CREATE TABLE customers (id int PRIMARY KEY, name text);
CREATE TABLE salaries (name text PRIMARY KEY, amount int);
INSERT INTO customers VALUES (1, 'Acme'), (2, 'Globex');
INSERT INTO salaries VALUES ('Ada', 120000), ('Linus', 110000);
-- Show audit lines in your own session too, not only in the server log.
ALTER DATABASE lesson_pgaudit SET pgaudit.log_client = on;
ALTER DATABASE lesson_pgaudit SET pgaudit.log_level = notice;
ALTER DATABASE lesson_pgaudit SET pgaudit.role = 'pgaudit_auditor';
```

```view
SELECT 'audited for every session' AS what, coalesce(string_agg(s, ', '), '(none)') AS details
FROM pg_db_role_setting, unnest(setconfig) s
WHERE setrole = 0 AND setdatabase = (SELECT oid FROM pg_database WHERE datname = current_database()) AND s LIKE 'pgaudit.log=%'
UNION ALL
SELECT 'table ' || table_name, string_agg(privilege_type, ', ' ORDER BY privilege_type)
FROM information_schema.role_table_grants WHERE grantee = 'pgaudit_auditor' GROUP BY table_name;
```

Rules like PCI DSS for card payments, HIPAA for health data, and SOC 2 for many software companies require an **audit trail**: a record of who read or changed sensitive data, and when. Postgres can log every statement with `log_statement = 'all'`, but that's a firehose. It doesn't say which tables a statement touched, and it misses statements run inside functions.

**pgaudit** is an extension that writes clear, structured audit lines. There are two ways to choose what's logged:

**Session auditing** logs whole classes of statement, set with `pgaudit.log`:

| Class | Logs |
|---|---|
| `read` | `SELECT`, `COPY ... TO` |
| `write` | `INSERT`, `UPDATE`, `DELETE`, `TRUNCATE`, `COPY ... FROM` |
| `ddl` | `CREATE`, `ALTER`, `DROP` |
| `role` | `GRANT`, `REVOKE`, `CREATE ROLE` |
| `all` | All of the above, and more |

**Object auditing** logs only statements touching specific tables. You pick an **auditor role**, `pgaudit.role`, and grant it privileges on the tables to watch. pgaudit logs any statement, by anyone, that uses a privilege the auditor holds. Granting `SELECT ON salaries` to the auditor means "log every read of salaries". The auditor role never logs in. It's only a list.

Each audit line looks like this:

```
AUDIT: OBJECT,1,1,READ,SELECT,TABLE,public.salaries,SELECT * FROM salaries,<not logged>
```

That's the kind of audit, statement numbers, class, command, object type, object name, and the statement itself.

**Where the lines go.** Normally into the server log, where a log system collects them. This playground also sends them to your own session as notices (`pgaudit.log_client`), so you can see them.

**Where it came from.** pgaudit is a third-party extension. It started at 2ndQuadrant in 2014, and Crunchy Data reworked it into today's version for Postgres 9.5 (2016). It was built to meet government and financial audit standards, and most hosted Postgres services offer it. It has to be loaded at server start, through `shared_preload_libraries`, which this course's server does.

**The Live View** shows what's audited in this playground: session classes saved for every session, and the tables the auditor role watches.

## Try it

Session auditing for writes, in this session only. The `INSERT` produces an `AUDIT: SESSION,...,WRITE,INSERT` notice. The `SELECT` doesn't, because reads aren't being audited.

```sql
SET pgaudit.log = 'write';
INSERT INTO customers VALUES (3, 'Initech');
SELECT count(*) FROM customers;
```

Add reads. Now the `SELECT` is logged too. Then turn session auditing back off.

```sql
SET pgaudit.log = 'read, write';
SELECT count(*) FROM customers;
RESET pgaudit.log;
```

Object auditing: watch table `customers` by granting the auditor role `SELECT` on it. A read of `customers` now logs an `AUDIT: OBJECT` line naming the table. A read of `salaries` logs nothing, because the auditor doesn't watch it yet. The Live View lists the watched table.

```sql
GRANT SELECT ON customers TO pgaudit_auditor;
SELECT * FROM customers WHERE id = 1;
SELECT * FROM salaries;
```

## Challenge

Table `salaries` is sensitive. Audit every read **and** every change to it, `SELECT`, `INSERT`, `UPDATE`, and `DELETE`, by anyone, using object auditing.

Hint: The last step's GRANT, with the four privileges, on salaries.

```solution
GRANT SELECT, INSERT, UPDATE, DELETE ON salaries TO pgaudit_auditor;
```

```check
SELECT bool_and(has_table_privilege('pgaudit_auditor', 'salaries', p))
FROM unnest(ARRAY['SELECT', 'INSERT', 'UPDATE', 'DELETE']) p;
```
