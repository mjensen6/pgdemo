-- Delete old rows from a big table in batches, committing after each one.
--
-- Same idea as batched-update.sql: short locks, and vacuum can keep up.
-- For data you drop by date every month, partitioning is better still:
-- dropping a partition is instant.
--
-- WRITES: deletes rows and commits as it goes. Run it outside a
-- transaction block.
--
-- Change: orders, the WHERE, and the batch size.

DO $$
DECLARE
  done bigint;
BEGIN
  LOOP
    DELETE FROM orders
    WHERE id IN (
      SELECT id FROM orders WHERE total < 1
      LIMIT 10000
    );
    GET DIAGNOSTICS done = ROW_COUNT;
    EXIT WHEN done = 0;
    COMMIT;
    RAISE NOTICE 'deleted % rows', done;
  END LOOP;
END $$;
