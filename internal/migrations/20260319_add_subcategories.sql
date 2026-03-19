CREATE TABLE IF NOT EXISTS subcategories (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    color       VARCHAR(50),
    created_at  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_subcategories_deleted_at ON subcategories(deleted_at);

ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS subcategory_id INTEGER;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'fk_subcategory_id' AND table_name = 'transactions'
    ) THEN
        ALTER TABLE transactions
            ADD CONSTRAINT fk_subcategory_id
            FOREIGN KEY (subcategory_id) REFERENCES subcategories(id) ON DELETE SET NULL;
    END IF;
END $$;
