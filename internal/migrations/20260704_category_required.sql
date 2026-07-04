-- Category is a required field on every transaction (M3).
-- This file must stay idempotent: the migration runner re-executes every
-- file on each boot.

-- Backfill: any transaction without a category moves to a fallback
-- 'Uncategorized' category (created only if needed).
DO $$
    DECLARE
        fallback_id INTEGER;
    BEGIN
        IF EXISTS (SELECT 1 FROM transactions WHERE category_id IS NULL) THEN
            SELECT id INTO fallback_id
            FROM categories
            WHERE name = 'Uncategorized' AND deleted_at IS NULL
            LIMIT 1;

            IF fallback_id IS NULL THEN
                INSERT INTO categories (name, description)
                VALUES ('Uncategorized', 'Fallback for transactions created before category became required')
                RETURNING id INTO fallback_id;
            END IF;

            UPDATE transactions SET category_id = fallback_id WHERE category_id IS NULL;
        END IF;
    END;
$$;

ALTER TABLE transactions ALTER COLUMN category_id SET NOT NULL;

-- The FK was ON DELETE SET NULL, which would now violate NOT NULL when a
-- category row is hard-deleted. RESTRICT makes the dependency explicit.
DO $$
    BEGIN
        IF EXISTS (
            SELECT 1
            FROM information_schema.referential_constraints
            WHERE constraint_name = 'fk_category_id'
              AND delete_rule = 'SET NULL'
        ) THEN
            ALTER TABLE transactions DROP CONSTRAINT fk_category_id;
            ALTER TABLE transactions ADD CONSTRAINT fk_category_id
                FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT;
        END IF;
    END;
$$;
