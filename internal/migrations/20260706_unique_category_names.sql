-- D6: category names must be unique (case- and whitespace-insensitive),
-- mirroring the locations pattern. Duplicate live categories are merged
-- first: transactions move to the oldest duplicate, the rest are
-- soft-deleted.

DO $$
    DECLARE
        dup RECORD;
    BEGIN
        FOR dup IN
            SELECT min(id) AS keeper, array_agg(id) AS ids
            FROM categories
            WHERE deleted_at IS NULL
            GROUP BY lower(btrim(name))
            HAVING count(*) > 1
        LOOP
            UPDATE transactions
            SET category_id = dup.keeper
            WHERE category_id = ANY(dup.ids) AND category_id <> dup.keeper;

            UPDATE categories
            SET deleted_at = CURRENT_TIMESTAMP
            WHERE id = ANY(dup.ids) AND id <> dup.keeper;
        END LOOP;
    END;
$$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_categories_normalized_name
    ON categories (lower(btrim(name)))
    WHERE deleted_at IS NULL;
