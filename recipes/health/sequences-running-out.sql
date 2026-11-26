-- Sequences, and the columns they fill, closest to their maximum.
--
-- A serial column is an int, which stops at 2,147,483,647, even though its
-- sequence could count far higher. When it runs out, every INSERT fails.
-- This checks each sequence against the smaller of its own maximum and its
-- column's type. The fix is ALTER TABLE ... ALTER COLUMN id TYPE bigint,
-- which rewrites the table, so plan it before you need it. Read-only.

SELECT s.schemaname || '.' || s.sequencename AS sequence,
       d.refobjid::regclass || '.' || a.attname AS column_name,
       format_type(a.atttypid, a.atttypmod) AS column_type,
       s.last_value,
       round(100.0 * s.last_value / least(s.max_value,
         CASE a.atttypid WHEN 'int2'::regtype THEN 32767
                         WHEN 'int4'::regtype THEN 2147483647
                         ELSE 9223372036854775807 END), 1) AS percent_used
FROM pg_sequences s
LEFT JOIN pg_depend d ON d.objid = format('%I.%I', s.schemaname, s.sequencename)::regclass
                     AND d.classid = 'pg_class'::regclass AND d.refclassid = 'pg_class'::regclass
                     AND d.deptype IN ('a', 'i')
LEFT JOIN pg_attribute a ON a.attrelid = d.refobjid AND a.attnum = d.refobjsubid
WHERE s.last_value IS NOT NULL
ORDER BY percent_used DESC NULLS LAST;
