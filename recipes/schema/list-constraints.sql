-- Every constraint in the database, written out as SQL.
--
-- Primary keys, unique, foreign keys, checks, exclusions, and NOT NULL
-- (stored as constraints since Postgres 18). NOT VALID ones were added
-- without checking existing rows. Read-only.

SELECT conrelid::regclass AS table_name,
       conname AS constraint_name,
       CASE contype WHEN 'p' THEN 'primary key' WHEN 'u' THEN 'unique'
                    WHEN 'f' THEN 'foreign key' WHEN 'c' THEN 'check'
                    WHEN 'x' THEN 'exclusion'   WHEN 'n' THEN 'not null'
                    ELSE contype::text END AS kind,
       NOT convalidated AS not_valid,
       pg_get_constraintdef(oid) AS definition
FROM pg_constraint
WHERE connamespace NOT IN ('pg_catalog'::regnamespace, 'information_schema'::regnamespace)
  AND conrelid <> 0
ORDER BY conrelid::regclass::text, contype, conname;
