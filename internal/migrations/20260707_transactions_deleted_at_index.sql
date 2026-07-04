-- With AutoMigrate removed (D4), the SQL migrations are the only schema
-- owner. This index previously came from GORM's soft-delete tag; keeping it
-- here makes fresh databases match existing ones.
CREATE INDEX IF NOT EXISTS idx_transactions_deleted_at ON transactions(deleted_at);
