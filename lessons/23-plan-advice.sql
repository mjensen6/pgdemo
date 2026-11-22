-- 23 pg_plan_advice (PG19): steer the query planner
LOAD 'pg_plan_advice';
CREATE TABLE a AS SELECT i AS id FROM generate_series(1, 10000) i;
CREATE TABLE b AS SELECT i AS id FROM generate_series(1, 10000) i;
CREATE INDEX b_id ON b (id);
ANALYZE a, b;
-- See the plan, written as advice you could reuse.
EXPLAIN (COSTS OFF, PLAN_ADVICE) SELECT * FROM a JOIN b ON a.id = b.id WHERE a.id < 100;
-- Now tell the planner to use a hash join and a sequential scan on b.
SET pg_plan_advice.advice = 'HASH_JOIN(b) SEQ_SCAN(b)';
EXPLAIN (COSTS OFF) SELECT * FROM a JOIN b ON a.id = b.id WHERE a.id < 100;
RESET pg_plan_advice.advice;
