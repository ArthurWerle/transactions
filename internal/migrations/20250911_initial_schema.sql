DO $$
    BEGIN
        IF NOT EXISTS (
            SELECT 1 FROM pg_type WHERE typname = 'transaction_type'
        ) THEN
            CREATE TYPE transaction_type AS ENUM ('income', 'expense');
        END IF;
    END;
$$;

DO $$
    BEGIN
        IF NOT EXISTS (
            SELECT 1 FROM pg_type WHERE typname = 'transaction_subtype'
        ) THEN
            CREATE TYPE transaction_subtype AS ENUM ('salary', 'profits', 'pro-labore');
        END IF;
    END;
$$;

CREATE TABLE IF NOT EXISTS transactions (
    id SERIAL PRIMARY KEY,
    migrated_id INTEGER,
    is_recurring BOOLEAN NOT NULL DEFAULT FALSE,
    category_id INTEGER,
    amount DECIMAL(12, 2) NOT NULL,
    type transaction_type NOT NULL,
    subtype transaction_subtype,
    description TEXT,
    date TIMESTAMP WITH TIME ZONE,
    frequency VARCHAR(50),
    start_date DATE,
    end_date DATE,
    created_by_id INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Add foreign key constraint if categories table exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'categories') THEN
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.table_constraints
            WHERE table_name = 'transactions' AND constraint_type = 'FOREIGN KEY'
        ) THEN
            ALTER TABLE transactions ADD CONSTRAINT fk_category_id
                FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL;
        END IF;
    END IF;
END $$;