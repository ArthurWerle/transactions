-- D1: enforce the recurring-transaction invariant at the database level.
-- Repairs run first (including soft-deleted rows, since CHECK constraints
-- apply to every row), then the constraints are added.

-- One-off rows must not carry schedule fields (legacy pre-F1 web
-- installments were stored this way; reports always counted them as
-- one-offs, so clearing the fields preserves reported history).
UPDATE transactions
SET start_date = NULL, end_date = NULL, frequency = NULL
WHERE is_recurring = false
  AND (start_date IS NOT NULL OR end_date IS NOT NULL OR frequency IS NOT NULL);

-- One-off rows must be dated; fall back to their creation time.
UPDATE transactions
SET date = created_at
WHERE is_recurring = false AND date IS NULL;

-- Recurring rows must have a start; derive from date or creation time.
UPDATE transactions
SET start_date = COALESCE(date::date, created_at::date)
WHERE is_recurring = true AND start_date IS NULL;

-- Recurring rows are schedules, not dated events.
UPDATE transactions
SET date = NULL
WHERE is_recurring = true AND date IS NOT NULL;

-- Recurring rows always carry the (only) frequency.
UPDATE transactions
SET frequency = 'monthly'
WHERE is_recurring = true AND frequency IS NULL;

-- A schedule cannot end before it starts; collapse to a single month.
UPDATE transactions
SET end_date = start_date
WHERE is_recurring = true AND end_date IS NOT NULL AND end_date < start_date;

DO $$
    BEGIN
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint WHERE conname = 'chk_transactions_shape'
        ) THEN
            ALTER TABLE transactions ADD CONSTRAINT chk_transactions_shape CHECK (
                (is_recurring AND date IS NULL AND start_date IS NOT NULL AND frequency IS NOT NULL)
                OR
                (NOT is_recurring AND date IS NOT NULL AND start_date IS NULL AND end_date IS NULL AND frequency IS NULL)
            );
        END IF;
    END;
$$;

DO $$
    BEGIN
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint WHERE conname = 'chk_transactions_dates'
        ) THEN
            ALTER TABLE transactions ADD CONSTRAINT chk_transactions_dates CHECK (
                end_date IS NULL OR start_date IS NULL OR end_date >= start_date
            );
        END IF;
    END;
$$;

-- The API has always rejected non-positive amounts; NOT VALID keeps boot
-- safe should some ancient row disagree, while still protecting new writes.
DO $$
    BEGIN
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint WHERE conname = 'chk_transactions_amount_positive'
        ) THEN
            ALTER TABLE transactions ADD CONSTRAINT chk_transactions_amount_positive
                CHECK (amount > 0) NOT VALID;
        END IF;
    END;
$$;
