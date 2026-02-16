-- Reset the categories id sequence to the current max id
-- This fixes duplicate key errors after migrating data from another database
SELECT setval('categories_id_seq', (SELECT COALESCE(MAX(id), 1) FROM categories));
