-- Data Migration: From Old Transactions Service to New Transactions Service
-- This script migrates data from the old transactions and recurring_transactions tables
-- to the new unified transactions table with enum types.
--
-- Type Mapping:
-- type_id 1 (Salary) → type: 'income', subtype: 'salary'
-- type_id 2 (income) → type: 'income', subtype: NULL
-- type_id 3 (expense) → type: 'expense', subtype: NULL

-- Create foreign server for the old database
CREATE SERVER IF NOT EXISTS old_db_server
  FOREIGN DATA WRAPPER postgres_fdw
  OPTIONS (host 'financer-services-postgres-1', dbname 'financer', port '5432');

-- Create user mapping (adjust user/password if needed)
CREATE USER MAPPING IF NOT EXISTS FOR postgres
  SERVER old_db_server
  OPTIONS (user 'admin', password 'nimda');

-- Import schema from old database
IMPORT FOREIGN SCHEMA public
  FROM SERVER old_db_server
  INTO old_db_schema;

-- Migrate regular transactions
INSERT INTO transactions (
  migrated_id,
  is_recurring,
  category_id,
  amount,
  type,
  subtype,
  description,
  date,
  frequency,
  start_date,
  end_date,
  created_by_id,
  created_at,
  updated_at
)
SELECT
  ot.id,
  FALSE,
  ot.category_id,
  ot.amount,
  CASE
    WHEN ot.type_id = 1 THEN 'income'::transaction_type
    WHEN ot.type_id = 2 THEN 'income'::transaction_type
    WHEN ot.type_id = 3 THEN 'expense'::transaction_type
  END as type,
  CASE
    WHEN ot.type_id = 1 THEN 'salary'::transaction_subtype
    ELSE NULL
  END as subtype,
  ot.description,
  ot.date,
  NULL,
  NULL,
  NULL,
  1 as created_by_id,
  ot.created_at,
  ot.updated_at
FROM old_db_schema.transactions ot;

-- Migrate recurring transactions
INSERT INTO transactions (
  migrated_id,
  is_recurring,
  category_id,
  amount,
  type,
  subtype,
  description,
  date,
  frequency,
  start_date,
  end_date,
  created_by_id,
  created_at,
  updated_at
)
SELECT
  ort.id,
  TRUE,
  ort.category_id,
  ort.amount,
  CASE
    WHEN ort.type_id = 1 THEN 'income'::transaction_type
    WHEN ort.type_id = 2 THEN 'income'::transaction_type
    WHEN ort.type_id = 3 THEN 'expense'::transaction_type
  END as type,
  CASE
    WHEN ort.type_id = 1 THEN 'salary'::transaction_subtype
    ELSE NULL
  END as subtype,
  ort.description,
  NULL as date,
  ort.frequency,
  ort.start_date,
  ort.end_date,
  1 as created_by_id,
  ort.created_at,
  ort.updated_at
FROM old_db_schema.recurring_transactions ort;

-- Clean up foreign data wrapper (optional - uncomment when done)
-- DROP USER MAPPING FOR postgres SERVER old_db_server;
-- DROP SERVER old_db_server;
-- DROP SCHEMA old_db_schema;
