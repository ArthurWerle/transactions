select * from transactions_v2 where is_recurring = TRUE;

SELECT setval(
               pg_get_serial_sequence('transactions_v2', 'id'),
               COALESCE((SELECT MAX(id) FROM transactions_v2), 0) + 1,
               false
       );


INSERT INTO transactions_v2 (
    migrated_id, category_id, amount, type, description,
    frequency, start_date, end_date,
    created_at, updated_at, is_recurring
)
SELECT
    r.id,
    r.category_id,
    r.amount,
    LOWER(tp.name)::transaction_type AS type,
    r.description,
    r.frequency,
    r.start_date,
    r.end_date,
    r.created_at,
    r.updated_at,
    TRUE
FROM recurring_transactions r
         LEFT JOIN types tp ON r.type_id = tp.id;












CREATE OR REPLACE FUNCTION sync_transaction_to_v2()
    RETURNS trigger AS $$
BEGIN
    INSERT INTO transactions_v2 (
        migrated_id,
        is_recurring,
        category_id,
        amount,
        type,
        subtype,
        description,
        date,
        created_at,
        updated_at
    ) VALUES (
                 NEW.id,          -- not recurring
                 FALSE,
                 NEW.category_id,
                 NEW.amount,
                 'income',
                 NULL,
                 NEW.description,
                 NEW.date,
                 NEW.created_at,
                 NEW.updated_at
             );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_sync_transaction_to_v2
    AFTER INSERT ON transactions
    FOR EACH ROW
EXECUTE FUNCTION sync_transaction_to_v2();






CREATE OR REPLACE FUNCTION sync_recurring_to_v2()
    RETURNS trigger AS $$
BEGIN
    INSERT INTO transactions_v2 (
        migrated_id,
        is_recurring,
        category_id,
        amount,
        type,
        subtype,
        description,
        frequency,
        start_date,
        end_date,
        created_at,
        updated_at
    ) VALUES (
                 NEW.id,
                 TRUE,
                 NEW.category_id,
                 NEW.amount,
                 'income',
                 NULL,
                 NEW.description,
                 NEW.frequency,
                 NEW.start_date,
                 NEW.end_date,
                 NEW.created_at,
                 NEW.updated_at
             );

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_sync_recurring_to_v2
    AFTER INSERT ON recurring_transactions
    FOR EACH ROW
EXECUTE FUNCTION sync_recurring_to_v2();






INSERT INTO transactions_v2 (
    migrated_id,
    is_recurring,
    category_id,
    amount,
    type,
    subtype,
    description,
    date,
    created_at,
    updated_at
)
SELECT
    t.id,            -- not recurring
    FALSE,
    t.category_id,
    t.amount,
    'income',          -- already an enum now
    null,
    t.description,
    t.date,
    t.created_at,
    t.updated_at
FROM transactions t
WHERE t.id > 980;


