-- Add prepaid_from_id column to track prepayment relationships
ALTER TABLE transactions
ADD COLUMN IF NOT EXISTS prepaid_from_id INTEGER REFERENCES transactions(id);

-- Create index for faster lookups
CREATE INDEX IF NOT EXISTS idx_transactions_prepaid_from_id ON transactions(prepaid_from_id);
