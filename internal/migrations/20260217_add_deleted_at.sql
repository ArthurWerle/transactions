-- Add deleted_at column to add a "soft delete" possibility.
ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
