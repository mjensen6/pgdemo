-- 08 UUIDs: random v4 and time-ordered v7 (PG18)
SELECT gen_random_uuid() AS v4_random;
-- v7 IDs sort by creation time, which keeps indexes compact.
SELECT uuidv7() AS v7 FROM generate_series(1, 3);
SELECT uuid_extract_timestamp(uuidv7()) AS created_at;
