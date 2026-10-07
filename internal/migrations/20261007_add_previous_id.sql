-- Links a recurring schedule to the one it replaced when its amount changed
-- from a given month on, so the price history stays traceable.
ALTER TABLE transactions
ADD COLUMN IF NOT EXISTS previous_id INTEGER REFERENCES transactions(id);

CREATE INDEX IF NOT EXISTS idx_transactions_previous_id ON transactions(previous_id);
