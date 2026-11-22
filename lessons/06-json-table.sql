-- 06 JSON_TABLE (PG17): turn JSON into rows and columns
SELECT * FROM JSON_TABLE(
  '[{"player": "Ada", "points": 21}, {"player": "Linus", "points": 14}]'::jsonb,
  '$[*]' COLUMNS (player text PATH '$.player', points int PATH '$.points')
) ORDER BY points DESC;
