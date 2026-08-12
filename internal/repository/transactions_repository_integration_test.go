package repository

// Integration tests for the SQL that unit tests can't reach: the shared
// period predicate, timezone bucketing, prepay exclusion, and the report
// queries. They run against a real Postgres when TEST_DATABASE_DSN is set
// (e.g. "host=127.0.0.1 user=transactions password=... dbname=transactions_test
// sslmode=disable") and skip otherwise, so `go test ./...` stays green
// without a database.

import (
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/ArthurWerle/transactions/internal/migrations"
	"github.com/ArthurWerle/transactions/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

var saoPaulo = mustLoadLocation("America/Sao_Paulo")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func setupIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set; skipping repository integration tests")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:         gormLogger.Default.LogMode(gormLogger.Silent),
		TranslateError: true,
	})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := migrations.RunMigrations(db, logger); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Every test starts from a clean slate.
	if err := db.Exec("TRUNCATE transactions, categories, subcategories, locations RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	return db
}

func seedCategory(t *testing.T, db *gorm.DB, name string) *model.Category {
	t.Helper()
	c := &model.Category{Name: name}
	if err := db.Create(c).Error; err != nil {
		t.Fatalf("failed to seed category: %v", err)
	}
	return c
}

func seedOneOff(t *testing.T, db *gorm.DB, catID uint, txType string, amount float64, date time.Time) *model.Transaction {
	t.Helper()
	tx := &model.Transaction{CategoryID: catID, Type: txType, Amount: amount, Date: &date}
	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("failed to seed one-off: %v", err)
	}
	return tx
}

func seedRecurring(t *testing.T, db *gorm.DB, catID uint, txType string, amount float64, start time.Time, end *time.Time) *model.Transaction {
	t.Helper()
	monthly := "monthly"
	tx := &model.Transaction{IsRecurring: true, CategoryID: catID, Type: txType, Amount: amount, StartDate: &start, EndDate: end, Frequency: &monthly}
	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("failed to seed recurring: %v", err)
	}
	return tx
}

func seedSubcategory(t *testing.T, db *gorm.DB, name string) *model.Subcategory {
	t.Helper()
	s := &model.Subcategory{Name: name}
	if err := db.Create(s).Error; err != nil {
		t.Fatalf("failed to seed subcategory: %v", err)
	}
	return s
}

func TestIntegration_PeriodPredicateBoundaries(t *testing.T) {
	db := setupIntegrationDB(t)
	repo := NewTransactionsRepository(db, saoPaulo)
	cat := seedCategory(t, db, "Boundaries")

	// 14:00 BRT on July 3rd — must be inside a range ending July 3rd.
	seedOneOff(t, db, cat.ID, "expense", 80, time.Date(2026, 7, 3, 14, 0, 0, 0, saoPaulo))
	// First instant of July 4th — must be outside (half-open).
	seedOneOff(t, db, cat.ID, "expense", 999, time.Date(2026, 7, 4, 0, 0, 0, 0, saoPaulo))
	// Recurring schedule that ENDS on the range start day — still overlaps.
	seedRecurring(t, db, cat.ID, "expense", 50, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), datePtrAt(2026, 7, 1))
	// Recurring schedule starting after the range — must not match.
	seedRecurring(t, db, cat.ID, "expense", 70, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), nil)

	// Range July 1–3 inclusive → repository takes the exclusive end (July 4).
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, saoPaulo)
	endExclusive := time.Date(2026, 7, 4, 0, 0, 0, 0, saoPaulo)
	rows, err := repo.FindByDateRange(start, endExclusive)
	if err != nil {
		t.Fatalf("FindByDateRange: %v", err)
	}

	amounts := map[float64]bool{}
	for _, r := range rows {
		amounts[r.Amount] = true
	}
	if !amounts[80] {
		t.Error("same-day afternoon transaction must be included (end day is inclusive)")
	}
	if amounts[999] {
		t.Error("next-day transaction must be excluded (half-open upper bound)")
	}
	if !amounts[50] {
		t.Error("recurring schedule ending on the range start must overlap")
	}
	if amounts[70] {
		t.Error("recurring schedule starting after the range must not match")
	}
}

func TestIntegration_TimezoneMonthBucketing(t *testing.T) {
	db := setupIntegrationDB(t)
	repo := NewTransactionsRepository(db, saoPaulo)
	cat := seedCategory(t, db, "TZ")

	// 2026-08-01T01:00Z is still July 31st 22:00 in São Paulo.
	seedOneOff(t, db, cat.ID, "expense", 123, time.Date(2026, 8, 1, 1, 0, 0, 0, time.UTC))

	rows, err := repo.FindNonRecurringMonthlyTotalsByType()
	if err != nil {
		t.Fatalf("FindNonRecurringMonthlyTotalsByType: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(rows))
	}
	if rows[0].Month != 7 {
		t.Errorf("expected the transaction bucketed into July (BRT), got month %d", rows[0].Month)
	}
}

func TestIntegration_MonthlyFlowReport(t *testing.T) {
	db := setupIntegrationDB(t)
	repo := NewTransactionsRepository(db, saoPaulo)
	cat := seedCategory(t, db, "Flow")

	// Recurring 500/month Mar–May; one-off income 6000 in April; May empty
	// otherwise; February entirely empty (must still appear, zero-filled).
	seedRecurring(t, db, cat.ID, "expense", 500, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), datePtrAt(2026, 5, 31))
	seedOneOff(t, db, cat.ID, "income", 6000, time.Date(2026, 4, 10, 12, 0, 0, 0, saoPaulo))
	// A prepayment lump must be excluded from aggregates.
	lumpDate := time.Date(2026, 4, 11, 12, 0, 0, 0, saoPaulo)
	original := seedRecurring(t, db, cat.ID, "expense", 10, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), datePtrAt(2026, 1, 31))
	lump := &model.Transaction{CategoryID: cat.ID, Type: "expense", Amount: 1500, Date: &lumpDate, PrepaidFromID: &original.ID}
	if err := db.Create(lump).Error; err != nil {
		t.Fatalf("failed to seed prepay lump: %v", err)
	}

	rows, err := repo.FindMonthlyFlow(
		time.Date(2026, 2, 1, 0, 0, 0, 0, saoPaulo),
		time.Date(2026, 5, 1, 0, 0, 0, 0, saoPaulo),
	)
	if err != nil {
		t.Fatalf("FindMonthlyFlow: %v", err)
	}
	if len(rows) != 4 {
		t.Fatalf("expected 4 months (Feb–May), got %d", len(rows))
	}

	if rows[0].Income != 0 || rows[0].Expense != 0 {
		t.Errorf("February must be zero-filled, got %+v", rows[0])
	}
	if rows[1].Expense != 500 {
		t.Errorf("March expense = %v, expected 500 (recurring)", rows[1].Expense)
	}
	if rows[2].Income != 6000 || rows[2].Expense != 500 {
		t.Errorf("April = %+v, expected income 6000 / expense 500 (lump excluded)", rows[2])
	}
	if rows[3].Expense != 500 {
		t.Errorf("May expense = %v, expected 500 (schedule ends May 31)", rows[3].Expense)
	}
}

func TestIntegration_CategoryMonthTotalsLabelDeleted(t *testing.T) {
	db := setupIntegrationDB(t)
	repo := NewTransactionsRepository(db, saoPaulo)
	live := seedCategory(t, db, "Groceries")
	gone := seedCategory(t, db, "Old Hobby")

	seedOneOff(t, db, live.ID, "expense", 300, time.Date(2026, 6, 10, 12, 0, 0, 0, saoPaulo))
	seedOneOff(t, db, gone.ID, "expense", 100, time.Date(2026, 6, 12, 12, 0, 0, 0, saoPaulo))
	if err := db.Delete(&model.Category{}, gone.ID).Error; err != nil {
		t.Fatalf("failed to soft-delete category: %v", err)
	}

	rows, err := repo.FindCategoryExpenseTotalsForMonth(time.Date(2026, 6, 1, 0, 0, 0, 0, saoPaulo))
	if err != nil {
		t.Fatalf("FindCategoryExpenseTotalsForMonth: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(rows))
	}
	// Ordered by total desc.
	if rows[0].CategoryName != "Groceries" || rows[0].Total != 300 {
		t.Errorf("unexpected first row: %+v", rows[0])
	}
	if rows[1].CategoryName != "Old Hobby (deleted)" || rows[1].Total != 100 {
		t.Errorf("expected labeled deleted category, got: %+v", rows[1])
	}
}

func seedLocation(t *testing.T, db *gorm.DB, name string) *model.Location {
	t.Helper()
	l := &model.Location{Name: name, NormalizedName: name}
	if err := db.Create(l).Error; err != nil {
		t.Fatalf("failed to seed location: %v", err)
	}
	return l
}

func seedOneOffLoc(t *testing.T, db *gorm.DB, catID uint, locID *uint, amount float64, date time.Time) {
	t.Helper()
	tx := &model.Transaction{CategoryID: catID, LocationID: locID, Type: "expense", Amount: amount, Date: &date}
	if err := db.Create(tx).Error; err != nil {
		t.Fatalf("failed to seed located one-off: %v", err)
	}
}

func TestIntegration_DailyExpenseTotalsForMonth(t *testing.T) {
	db := setupIntegrationDB(t)
	repo := NewTransactionsRepository(db, saoPaulo)
	cat := seedCategory(t, db, "Daily")

	// Two one-off expenses on June 3rd (BRT) → that day totals 150.
	seedOneOff(t, db, cat.ID, "expense", 100, time.Date(2026, 6, 3, 14, 0, 0, 0, saoPaulo))
	seedOneOff(t, db, cat.ID, "expense", 50, time.Date(2026, 6, 3, 18, 0, 0, 0, saoPaulo))
	// Income must not count toward expenses.
	seedOneOff(t, db, cat.ID, "income", 999, time.Date(2026, 6, 5, 12, 0, 0, 0, saoPaulo))
	// Recurring day-15 schedule active in June → attributed to June 15.
	seedRecurring(t, db, cat.ID, "expense", 200, time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), nil)
	// Recurring day-31 schedule → clamped to the last day of June (30th).
	seedRecurring(t, db, cat.ID, "expense", 300, time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC), nil)
	// Recurring starting after June must not appear.
	seedRecurring(t, db, cat.ID, "expense", 70, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), nil)
	// A prepayment lump is a cash record excluded from aggregates.
	lumpDate := time.Date(2026, 6, 20, 12, 0, 0, 0, saoPaulo)
	original := seedRecurring(t, db, cat.ID, "expense", 10, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), datePtrAt(2026, 1, 31))
	lump := &model.Transaction{CategoryID: cat.ID, Type: "expense", Amount: 1500, Date: &lumpDate, PrepaidFromID: &original.ID}
	if err := db.Create(lump).Error; err != nil {
		t.Fatalf("failed to seed prepay lump: %v", err)
	}

	days, count, err := repo.FindDailyExpenseTotalsForMonth(time.Date(2026, 6, 1, 0, 0, 0, 0, saoPaulo))
	if err != nil {
		t.Fatalf("FindDailyExpenseTotalsForMonth: %v", err)
	}
	if len(days) != 30 {
		t.Fatalf("June has 30 days (zero-filled), got %d", len(days))
	}

	byDay := map[int]float64{}
	var sum float64
	for _, d := range days {
		byDay[d.Day.Day()] = d.Total
		sum += d.Total
	}
	if byDay[3] != 150 {
		t.Errorf("June 3 total = %v, expected 150", byDay[3])
	}
	if byDay[15] != 200 {
		t.Errorf("June 15 total = %v, expected 200 (recurring day-15)", byDay[15])
	}
	if byDay[30] != 300 {
		t.Errorf("June 30 total = %v, expected 300 (recurring day-31 clamped)", byDay[30])
	}
	if sum != 650 {
		t.Errorf("daily sum = %v, expected 650 (income + prepay + future-recurring excluded)", sum)
	}
	// 2 one-off expenses + 2 active recurring expenses = 4.
	if count != 4 {
		t.Errorf("transaction_count = %d, expected 4", count)
	}
}

func TestIntegration_MerchantExpenseTotalsForMonth(t *testing.T) {
	db := setupIntegrationDB(t)
	repo := NewTransactionsRepository(db, saoPaulo)
	groceries := seedCategory(t, db, "Groceries")
	coffee := seedCategory(t, db, "Coffee")
	market := seedLocation(t, db, "Supermarket")
	cafe := seedLocation(t, db, "Cafe")

	// Supermarket: Groceries 300 (2 tx) + Coffee 10 (1 tx) → top category Groceries.
	seedOneOffLoc(t, db, groceries.ID, &market.ID, 100, time.Date(2026, 6, 3, 12, 0, 0, 0, saoPaulo))
	seedOneOffLoc(t, db, groceries.ID, &market.ID, 200, time.Date(2026, 6, 10, 12, 0, 0, 0, saoPaulo))
	seedOneOffLoc(t, db, coffee.ID, &market.ID, 10, time.Date(2026, 6, 11, 12, 0, 0, 0, saoPaulo))
	// Cafe: Coffee 40 (1 tx).
	seedOneOffLoc(t, db, coffee.ID, &cafe.ID, 40, time.Date(2026, 6, 5, 12, 0, 0, 0, saoPaulo))
	// No location → folds into "(none)".
	seedOneOffLoc(t, db, groceries.ID, nil, 30, time.Date(2026, 6, 7, 12, 0, 0, 0, saoPaulo))

	rows, err := repo.FindMerchantExpenseTotalsForMonth(time.Date(2026, 6, 1, 0, 0, 0, 0, saoPaulo))
	if err != nil {
		t.Fatalf("FindMerchantExpenseTotalsForMonth: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 merchants, got %d", len(rows))
	}
	// Ordered by total desc: Supermarket 310, Cafe 40, (none) 30.
	if rows[0].Name != "Supermarket" || rows[0].Total != 310 || rows[0].TransactionCount != 3 || rows[0].TopCategory != "Groceries" {
		t.Errorf("unexpected Supermarket row: %+v", rows[0])
	}
	if rows[1].Name != "Cafe" || rows[1].Total != 40 || rows[1].TransactionCount != 1 || rows[1].TopCategory != "Coffee" {
		t.Errorf("unexpected Cafe row: %+v", rows[1])
	}
	if rows[2].Name != "(none)" || rows[2].Total != 30 || rows[2].TransactionCount != 1 {
		t.Errorf("expected the (none) bucket last, got %+v", rows[2])
	}
}

func TestIntegration_DuplicateCategoryTranslatesError(t *testing.T) {
	db := setupIntegrationDB(t)
	repo := NewCategoryRepository(db)

	if err := repo.Create(&model.Category{Name: "Food"}); err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	err := repo.Create(&model.Category{Name: "  FOOD "})
	if err == nil {
		t.Fatal("expected duplicate-name create to fail")
	}
	if err != gorm.ErrDuplicatedKey {
		t.Errorf("expected gorm.ErrDuplicatedKey, got %v", err)
	}
}

func datePtrAt(year int, month time.Month, day int) *time.Time {
	d := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &d
}

// TestIntegration_UpdatePersistsSubcategoryChange guards against the GORM
// auto-save-associations gotcha: FindByID preloads the belongs-to Subcategory,
// and a plain Save would upsert that stale association and overwrite
// subcategory_id back to its old value. Update must persist the new FK.
func TestIntegration_UpdatePersistsSubcategoryChange(t *testing.T) {
	db := setupIntegrationDB(t)
	repo := NewTransactionsRepository(db, saoPaulo)
	cat := seedCategory(t, db, "Groceries")
	subA := seedSubcategory(t, db, "Old sub")
	subB := seedSubcategory(t, db, "New sub")
	// Stable expected values, never handed out as pointers: a buggy Update
	// that clobbers the FK writes *through* the pointer we pass, so any uint
	// used as a pointer target can be mutated out from under us.
	wantOld, wantNew := subA.ID, subB.ID

	tx := seedOneOff(t, db, cat.ID, "expense", 42, time.Date(2026, 7, 3, 14, 0, 0, 0, saoPaulo))
	oldPtr := subA.ID
	tx.SubcategoryID = &oldPtr
	if err := repo.Update(tx); err != nil {
		t.Fatalf("initial Update: %v", err)
	}

	// Reload the way the handler does — this preloads the stale Subcategory
	// association that used to clobber the foreign key on Save.
	loaded, err := repo.FindByID(tx.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if loaded.SubcategoryID == nil || *loaded.SubcategoryID != wantOld {
		t.Fatalf("setup: expected subcategory %d, got %v", wantOld, loaded.SubcategoryID)
	}

	// Change only the foreign key, leaving the preloaded association untouched.
	newPtr := subB.ID
	loaded.SubcategoryID = &newPtr
	if err := repo.Update(loaded); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Ground truth: the persisted column, independent of any preload.
	var rawFK *uint
	if err := db.Raw("SELECT subcategory_id FROM transactions WHERE id = ?", tx.ID).Scan(&rawFK).Error; err != nil {
		t.Fatalf("raw subcategory_id read: %v", err)
	}
	if rawFK == nil {
		t.Fatalf("subcategory change did not persist in DB: want %d, got NULL", wantNew)
	}
	if *rawFK != wantNew {
		t.Fatalf("subcategory change did not persist in DB: want %d, got %d", wantNew, *rawFK)
	}

	after, err := repo.FindByID(tx.ID)
	if err != nil {
		t.Fatalf("FindByID after update: %v", err)
	}
	if after.SubcategoryID == nil || *after.SubcategoryID != wantNew {
		t.Fatalf("subcategory change did not persist: want %d, got %v", wantNew, after.SubcategoryID)
	}
	if after.Subcategory == nil || after.Subcategory.Name != "New sub" {
		t.Errorf("preloaded subcategory should reflect the new value, got %+v", after.Subcategory)
	}
}
