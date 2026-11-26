-- Who can do what on each table, beyond the owner.
--
-- Lists every grant on tables and views, including those given to PUBLIC
-- (everyone). Owners and superusers can do everything and aren't listed.
-- Read-only.

SELECT c.oid::regclass AS table_name,
       CASE a.grantee WHEN 0 THEN 'PUBLIC' ELSE a.grantee::regrole::text END AS grantee,
       string_agg(a.privilege_type, ', ' ORDER BY a.privilege_type) AS privileges,
       a.grantor::regrole AS granted_by
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
CROSS JOIN LATERAL aclexplode(c.relacl) a
WHERE c.relkind IN ('r', 'p', 'v', 'm', 'f')
  AND n.nspname NOT IN ('pg_catalog', 'information_schema')
  AND a.grantee <> c.relowner
GROUP BY c.oid, a.grantee, a.grantor
ORDER BY 1, 2;
