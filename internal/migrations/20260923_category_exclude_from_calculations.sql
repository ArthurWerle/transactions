-- Categories flagged with exclude_from_calculations (e.g. one-off purchases)
-- are left out of every aggregate: averages, reports and percentages. Their
-- transactions still show up in listings.
ALTER TABLE categories
    ADD COLUMN IF NOT EXISTS exclude_from_calculations BOOLEAN NOT NULL DEFAULT false;
