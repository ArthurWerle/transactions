-- Monthly is the only supported recurrence frequency (F2).
-- This file must stay idempotent: the migration runner re-executes every
-- file on each boot.

DO $$
    BEGIN
        IF NOT EXISTS (
            SELECT 1 FROM pg_type WHERE typname = 'transaction_frequency'
        ) THEN
            CREATE TYPE transaction_frequency AS ENUM ('monthly');
        END IF;
    END;
$$;

-- Normalize legacy values before the cast. Every calculation always assumed
-- monthly, so this makes stored data match reported behavior.
UPDATE transactions
SET frequency = 'monthly'
WHERE frequency IS NOT NULL
  AND frequency::text <> 'monthly';

DO $$
    BEGIN
        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_name = 'transactions'
              AND column_name = 'frequency'
              AND udt_name <> 'transaction_frequency'
        ) THEN
            ALTER TABLE transactions
                ALTER COLUMN frequency TYPE transaction_frequency
                USING frequency::text::transaction_frequency;
        END IF;
    END;
$$;
