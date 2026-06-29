DO $$
    BEGIN
        IF NOT EXISTS (
            SELECT 1 FROM pg_type WHERE typname = 'transaction_origin'
        ) THEN
            CREATE TYPE transaction_origin AS ENUM ('web', 'api', 'mcp');
        END IF;
    END;
$$;

ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS origin transaction_origin NOT NULL DEFAULT 'api';

COMMENT ON COLUMN transactions.subtype IS 'Deprecated: do not use. Will be removed in a future migration.';
