CREATE TABLE IF NOT EXISTS locations (
    id              SERIAL PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    normalized_name VARCHAR(255) NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at      TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_locations_normalized_name ON locations(normalized_name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_locations_deleted_at ON locations(deleted_at);

ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS location_id INTEGER;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'fk_location_id' AND table_name = 'transactions'
    ) THEN
        ALTER TABLE transactions
            ADD CONSTRAINT fk_location_id
            FOREIGN KEY (location_id) REFERENCES locations(id) ON DELETE SET NULL;
    END IF;
END $$;
