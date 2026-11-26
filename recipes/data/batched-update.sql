-- Update a big table in batches, committing after each one.
--
-- One UPDATE of millions of rows holds its row locks until the end, and
-- leaves every replaced row dead at once. Batches keep locks short and let
-- vacuum keep up. Run it outside a transaction block: a DO block can only
-- COMMIT then.
--
-- WRITES: it updates and commits as it goes. If it stops, run it again:
-- the WHERE skips rows already done.
--
-- Change: orders, the SET, the WHERE that finds rows still to do, and
-- the batch size.

DO $$
DECLARE
  done bigint;
BEGIN
  LOOP
    UPDATE orders SET status = 'paid'
    WHERE id IN (
      SELECT id FROM orders WHERE status IS NULL
      LIMIT 10000
      FOR UPDATE SKIP LOCKED
    );
    GET DIAGNOSTICS done = ROW_COUNT;
    EXIT WHEN done = 0;
    COMMIT;
    RAISE NOTICE 'updated % rows', done;
    -- PERFORM pg_sleep(0.1);  -- uncomment to give other work a gap
  END LOOP;
END $$;
