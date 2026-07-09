package service

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/ArthurWerle/transactions/internal/model"
	"github.com/ArthurWerle/transactions/internal/repository"
)

type mockTransactionsRepository struct {
	transactions           map[uint]*model.Transaction
	created                []*model.Transaction
	nonRecurringSums       []repository.CategoryExpenseSummary
	recurringExpenses      []repository.RecurringCategoryExpense
	earliestDate           *time.Time
	monthlyTotals          []repository.MonthlyTypeTotal
	recurringByType        []repository.RecurringTypeTransaction
	incomeTotal            float64
	recurringIncomes       []repository.RecurringAmount
	monthlyFlow            []repository.MonthlyFlowRow
	categoryFlow           []repository.CategoryMonthlyFlowRow
	categoryMonthTotals    []repository.CategoryMonthTotal
	subcategoryMonthTotals []repository.SubcategoryMonthTotal
	locationMonthTotals    []repository.LocationMonthTotal
}

func newMockRepository() *mockTransactionsRepository {
	return &mockTransactionsRepository{
		transactions: make(map[uint]*model.Transaction),
	}
}

func (m *mockTransactionsRepository) Create(transaction *model.Transaction) error {
	if transaction.ID == 0 {
		transaction.ID = uint(len(m.transactions) + 1)
	}
	m.transactions[transaction.ID] = transaction
	m.created = append(m.created, transaction)
	return nil
}

func (m *mockTransactionsRepository) FindByID(id uint) (*model.Transaction, error) {
	t, ok := m.transactions[id]
	if !ok {
		return nil, errors.New("record not found")
	}
	return t, nil
}

func (m *mockTransactionsRepository) FindAll(limit, offset int) ([]model.Transaction, error) {
	return nil, nil
}

func (m *mockTransactionsRepository) CountAll() (int64, error) {
	return int64(len(m.transactions)), nil
}

func (m *mockTransactionsRepository) FindAllWithFilters(currentMonth bool, categoryIDs []uint, searchQuery string, startDate, endDate *time.Time, transactionType string, limit, offset int) ([]model.Transaction, error) {
	return nil, nil
}

func (m *mockTransactionsRepository) CountAllWithFilters(currentMonth bool, categoryIDs []uint, searchQuery string, startDate, endDate *time.Time, transactionType string) (int64, error) {
	return 0, nil
}

func (m *mockTransactionsRepository) FindBiggest(month, year int) ([]model.Transaction, error) {
	return nil, nil
}

func (m *mockTransactionsRepository) FindLatest() ([]model.Transaction, error) {
	return nil, nil
}

func (m *mockTransactionsRepository) Update(transaction *model.Transaction) error {
	m.transactions[transaction.ID] = transaction
	return nil
}

func (m *mockTransactionsRepository) Delete(id uint) error {
	return nil
}

func (m *mockTransactionsRepository) FindByDateRange(startDate, endDate time.Time) ([]model.Transaction, error) {
	return nil, nil
}

func (m *mockTransactionsRepository) FindMonthlyFlow(startMonth, endMonth time.Time) ([]repository.MonthlyFlowRow, error) {
	return m.monthlyFlow, nil
}

func (m *mockTransactionsRepository) FindCategoryMonthlyFlow(startMonth, endMonth time.Time, categoryIDs []uint) ([]repository.CategoryMonthlyFlowRow, error) {
	return m.categoryFlow, nil
}

func (m *mockTransactionsRepository) FindCategoryExpenseTotalsForMonth(month time.Time) ([]repository.CategoryMonthTotal, error) {
	return m.categoryMonthTotals, nil
}

func (m *mockTransactionsRepository) FindSubcategoryExpenseTotalsForMonth(month time.Time) ([]repository.SubcategoryMonthTotal, error) {
	return m.subcategoryMonthTotals, nil
}

func (m *mockTransactionsRepository) FindLocationExpenseTotalsForMonth(month time.Time) ([]repository.LocationMonthTotal, error) {
	return m.locationMonthTotals, nil
}

func (m *mockTransactionsRepository) FindByPrepaidID(prepaidID uint) (*model.Transaction, error) {
	return nil, nil
}

func (m *mockTransactionsRepository) FindEarliestDate(startDate, endDate *time.Time) (*time.Time, error) {
	return m.earliestDate, nil
}

func (m *mockTransactionsRepository) FindExpenseSummaryByCategory(startDate, endDate *time.Time) ([]repository.CategoryExpenseSummary, error) {
	return m.nonRecurringSums, nil
}

func (m *mockTransactionsRepository) FindRecurringExpensesInRange(startDate, endDate *time.Time) ([]repository.RecurringCategoryExpense, error) {
	return m.recurringExpenses, nil
}

func (m *mockTransactionsRepository) FindIncomeTotalInRange(startDate, endDate *time.Time) (float64, error) {
	return m.incomeTotal, nil
}

func (m *mockTransactionsRepository) FindRecurringIncomesInRange(startDate, endDate *time.Time) ([]repository.RecurringAmount, error) {
	return m.recurringIncomes, nil
}

func (m *mockTransactionsRepository) FindCurrentMonthTotalByType(transactionType string) (float64, error) {
	return 0, nil
}

func (m *mockTransactionsRepository) FindCurrentMonthTotalByTypeAndCategory(transactionType string, categoryID uint) (float64, error) {
	return 0, nil
}

func (m *mockTransactionsRepository) FindNonRecurringMonthlyTotalsByType() ([]repository.MonthlyTypeTotal, error) {
	return m.monthlyTotals, nil
}

func (m *mockTransactionsRepository) FindRecurringTransactionSummaryByType() ([]repository.RecurringTypeTransaction, error) {
	return m.recurringByType, nil
}

func newTestService(repo repository.TransactionsRepository) TransactionsService {
	return NewTransactionsService(repo, time.UTC)
}

func monthsAgo(n int) time.Time {
	return MonthStart(time.Now().UTC()).AddDate(0, -n, 0)
}

// ---- month count tests ----

func TestInclusiveMonthCount(t *testing.T) {
	tests := []struct {
		name     string
		start    time.Time
		end      time.Time
		expected int
	}{
		{
			name:     "same month",
			start:    time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			expected: 1,
		},
		{
			name:     "consecutive months",
			start:    time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			expected: 2,
		},
		{
			name:     "5 months",
			start:    time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			expected: 5,
		},
		{
			name:     "across year boundary",
			start:    time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC),
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InclusiveMonthCount(tt.start, tt.end)
			if result != tt.expected {
				t.Errorf("InclusiveMonthCount(%v, %v) = %d, expected %d", tt.start, tt.end, result, tt.expected)
			}
		})
	}
}

func TestExclusiveMonthCount(t *testing.T) {
	tests := []struct {
		name     string
		start    time.Time
		end      time.Time
		expected int
	}{
		{
			name:     "same month yields 0 remaining",
			start:    time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			expected: 0,
		},
		{
			name:     "one month ahead yields 1 remaining",
			start:    time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			expected: 1,
		},
		{
			name:     "5 months ahead yields 5 remaining",
			start:    time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExclusiveMonthCount(tt.start, tt.end)
			if result != tt.expected {
				t.Errorf("ExclusiveMonthCount(%v, %v) = %d, expected %d", tt.start, tt.end, result, tt.expected)
			}
		})
	}
}

// ---- validateTransactionShape tests ----

func TestTransactionShapeValidation(t *testing.T) {
	now := time.Now()
	start := monthsAgo(3)
	end := monthsAgo(0).AddDate(0, 2, 0)
	monthly := MonthlyFrequency
	weekly := "weekly"

	tests := []struct {
		name    string
		tx      model.Transaction
		wantErr string
	}{
		{
			name: "valid one-off",
			tx:   model.Transaction{Date: &now, Amount: 10, Type: "expense", CategoryID: 1},
		},
		{
			name: "valid recurring",
			tx:   model.Transaction{IsRecurring: true, StartDate: &start, EndDate: &end, Amount: 10, Type: "expense", CategoryID: 1},
		},
		{
			name:    "missing category",
			tx:      model.Transaction{Date: &now, Amount: 10, Type: "expense"},
			wantErr: "category_id is required",
		},
		{
			name:    "non-positive amount",
			tx:      model.Transaction{Date: &now, Amount: 0, Type: "expense", CategoryID: 1},
			wantErr: "amount must be greater than 0",
		},
		{
			name:    "invalid type",
			tx:      model.Transaction{Date: &now, Amount: 10, Type: "transfer", CategoryID: 1},
			wantErr: "type must be either 'income' or 'expense'",
		},
		{
			name:    "recurring without start_date",
			tx:      model.Transaction{IsRecurring: true, EndDate: &end, Amount: 10, Type: "expense", CategoryID: 1},
			wantErr: "start_date",
		},
		{
			name:    "recurring with date",
			tx:      model.Transaction{IsRecurring: true, StartDate: &start, Date: &now, Amount: 10, Type: "expense", CategoryID: 1},
			wantErr: "must not have date",
		},
		{
			name:    "recurring with non-monthly frequency",
			tx:      model.Transaction{IsRecurring: true, StartDate: &start, Frequency: &weekly, Amount: 10, Type: "expense", CategoryID: 1},
			wantErr: "monthly",
		},
		{
			name:    "recurring with end before start",
			tx:      model.Transaction{IsRecurring: true, StartDate: &end, EndDate: &start, Frequency: &monthly, Amount: 10, Type: "expense", CategoryID: 1},
			wantErr: "end_date must not be before start_date",
		},
		{
			name:    "one-off without date",
			tx:      model.Transaction{Amount: 10, Type: "expense", CategoryID: 1},
			wantErr: "requires date",
		},
		{
			name:    "one-off with schedule fields",
			tx:      model.Transaction{Date: &now, EndDate: &end, Amount: 10, Type: "expense", CategoryID: 1},
			wantErr: "must not have start_date, end_date or frequency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			svc := newTestService(repo)
			tx := tt.tx
			err := svc.CreateTransaction(context.Background(), &tx)

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected transaction to be valid, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !errors.Is(err, ErrInvalidTransaction) {
				t.Errorf("error should wrap ErrInvalidTransaction, got %v", err)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q should mention %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestCreateTransaction_DefaultsRecurringFrequencyToMonthly(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	start := monthsAgo(1)
	tx := model.Transaction{IsRecurring: true, StartDate: &start, Amount: 10, Type: "expense", CategoryID: 1}
	if err := svc.CreateTransaction(context.Background(), &tx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx.Frequency == nil || *tx.Frequency != MonthlyFrequency {
		t.Errorf("expected frequency to default to monthly, got %v", tx.Frequency)
	}
}

func TestUpdateTransaction_NormalizesShape(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	now := time.Now()
	start := monthsAgo(2)
	end := monthsAgo(0)
	weekly := "weekly"

	// Legacy corrupted row: one-off carrying schedule fields (pre-F1 web data).
	legacy := model.Transaction{ID: 1, Date: &now, EndDate: &end, Frequency: &weekly, Amount: 10, Type: "expense", CategoryID: 1}
	if err := svc.UpdateTransaction(context.Background(), &legacy); err != nil {
		t.Fatalf("expected legacy row to be healed, got %v", err)
	}
	if legacy.EndDate != nil || legacy.Frequency != nil {
		t.Error("expected schedule fields to be cleared on non-recurring update")
	}

	// Flip to recurring: date must be cleared, frequency coerced to monthly.
	rec := model.Transaction{ID: 2, IsRecurring: true, Date: &now, StartDate: &start, Frequency: &weekly, Amount: 10, Type: "expense", CategoryID: 1}
	if err := svc.UpdateTransaction(context.Background(), &rec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Date != nil {
		t.Error("expected date to be cleared on recurring update")
	}
	if rec.Frequency == nil || *rec.Frequency != MonthlyFrequency {
		t.Errorf("expected frequency coerced to monthly, got %v", rec.Frequency)
	}

	// Flip to recurring without start_date must fail.
	bad := model.Transaction{ID: 3, IsRecurring: true, Date: &now, Amount: 10, Type: "expense", CategoryID: 1}
	if err := svc.UpdateTransaction(context.Background(), &bad); !errors.Is(err, ErrInvalidTransaction) {
		t.Errorf("expected ErrInvalidTransaction, got %v", err)
	}
}

// ---- GetAverageByType tests ----

func lastCompleteMonth() time.Time {
	return MonthStart(time.Now().UTC()).AddDate(0, -1, 0)
}

func TestGetAverageByType_PerTypeRanges(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	// income since 6 months ago, expenses only since 2 months ago: each type
	// must be averaged over its own range (F7).
	incomeMonth := monthsAgo(6)
	expenseMonth := monthsAgo(2)
	repo.monthlyTotals = []repository.MonthlyTypeTotal{
		{Type: "income", Year: incomeMonth.Year(), Month: int(incomeMonth.Month()), MonthlySum: 600},
		{Type: "expense", Year: expenseMonth.Year(), Month: int(expenseMonth.Month()), MonthlySum: 200},
	}

	result, err := svc.GetAverageByType(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	byType := make(map[string]float64)
	for _, r := range result {
		byType[r.TypeName] = r.Average
	}

	expectedIncome := 600.0 / float64(InclusiveMonthCount(incomeMonth, lastCompleteMonth()))
	expectedExpense := 200.0 / float64(InclusiveMonthCount(expenseMonth, lastCompleteMonth()))

	if math.Abs(byType["income"]-expectedIncome) > 0.001 {
		t.Errorf("income average = %v, expected %v", byType["income"], expectedIncome)
	}
	if math.Abs(byType["expense"]-expectedExpense) > 0.001 {
		t.Errorf("expense average = %v, expected %v", byType["expense"], expectedExpense)
	}
}

func TestGetAverageByType_ExcludesCurrentMonth(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	// One complete month of data plus a huge current-month total: the
	// in-progress month must not dilute or inflate the average.
	previous := monthsAgo(1)
	current := monthsAgo(0)
	repo.monthlyTotals = []repository.MonthlyTypeTotal{
		{Type: "expense", Year: previous.Year(), Month: int(previous.Month()), MonthlySum: 100},
		{Type: "expense", Year: current.Year(), Month: int(current.Month()), MonthlySum: 9999},
	}

	result, err := svc.GetAverageByType(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 type, got %d", len(result))
	}
	if result[0].Average != 100 {
		t.Errorf("expected average 100 (current month excluded), got %v", result[0].Average)
	}
}

func TestGetAverageByType_CurrentMonthOnlyFallback(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	current := monthsAgo(0)
	repo.monthlyTotals = []repository.MonthlyTypeTotal{
		{Type: "expense", Year: current.Year(), Month: int(current.Month()), MonthlySum: 300},
	}

	result, err := svc.GetAverageByType(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 type, got %d", len(result))
	}
	if result[0].Average != 300 {
		t.Errorf("expected current-month fallback average 300, got %v", result[0].Average)
	}
}

func TestGetAverageByType_RecurringExpansion(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	// Recurring $20/month over the last 3 complete months (ends last month).
	start := monthsAgo(3)
	end := monthsAgo(1)
	repo.recurringByType = []repository.RecurringTypeTransaction{
		{Type: "expense", Amount: 20, StartDate: &start, EndDate: &end},
	}

	result, err := svc.GetAverageByType(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 type, got %d", len(result))
	}

	// Total = $20 × 3 active months; range = start..lastComplete = 3 months.
	if math.Abs(result[0].Average-20) > 0.001 {
		t.Errorf("expected average 20, got %v", result[0].Average)
	}
}

func TestGetAverageByType_MixedRecurringAndNon(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	// Non-recurring $50 three months ago; recurring $20/month from 3 months
	// ago through last month (3 complete months = $60). Range = 3 months.
	oneOff := monthsAgo(3)
	recurringStart := monthsAgo(3)
	recurringEnd := monthsAgo(1)

	repo.monthlyTotals = []repository.MonthlyTypeTotal{
		{Type: "expense", Year: oneOff.Year(), Month: int(oneOff.Month()), MonthlySum: 50},
	}
	repo.recurringByType = []repository.RecurringTypeTransaction{
		{Type: "expense", Amount: 20, StartDate: &recurringStart, EndDate: &recurringEnd},
	}

	result, err := svc.GetAverageByType(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 type, got %d", len(result))
	}

	expected := (50.0 + 60.0) / 3.0
	if math.Abs(result[0].Average-expected) > 0.001 {
		t.Errorf("expected average %.4f, got %.4f", expected, result[0].Average)
	}
}

func TestGetAverageByType_WindowClipsOldMonths(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	// Spending 8 months ago is outside a 6-month trailing window (M1) but
	// inside a 12-month one.
	oldMonth := monthsAgo(8)
	recentMonth := monthsAgo(2)
	repo.monthlyTotals = []repository.MonthlyTypeTotal{
		{Type: "expense", Year: oldMonth.Year(), Month: int(oldMonth.Month()), MonthlySum: 900},
		{Type: "expense", Year: recentMonth.Year(), Month: int(recentMonth.Month()), MonthlySum: 300},
	}

	narrow, err := svc.GetAverageByType(context.Background(), 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only the recent month counts, averaged over its own in-window range.
	expectedNarrow := 300.0 / float64(InclusiveMonthCount(recentMonth, lastCompleteMonth()))
	if math.Abs(narrow[0].Average-expectedNarrow) > 0.001 {
		t.Errorf("window=6 average = %v, expected %v (old month excluded)", narrow[0].Average, expectedNarrow)
	}

	wide, err := svc.GetAverageByType(context.Background(), 12)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedWide := (900.0 + 300.0) / float64(InclusiveMonthCount(oldMonth, lastCompleteMonth()))
	if math.Abs(wide[0].Average-expectedWide) > 0.001 {
		t.Errorf("window=12 average = %v, expected %v (old month included)", wide[0].Average, expectedWide)
	}
}

func TestGetAverageByType_WindowClipsRecurringStart(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	// Open-ended recurring $100/month running for 20 months: a 6-month
	// window must average exactly $100, not dilute or accumulate.
	start := monthsAgo(20)
	repo.recurringByType = []repository.RecurringTypeTransaction{
		{Type: "expense", Amount: 100, StartDate: &start, EndDate: nil},
	}

	result, err := svc.GetAverageByType(context.Background(), 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 type, got %d", len(result))
	}
	if math.Abs(result[0].Average-100) > 0.001 {
		t.Errorf("expected average 100 over the window, got %v", result[0].Average)
	}
}

func TestGetAverageByType_EmptyData(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	result, err := svc.GetAverageByType(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil result for empty data, got %v", result)
	}
}

// ---- PrepayTransaction tests ----

func TestPrepayTransaction_StartedSchedule(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	// Started 6 months ago, ends 3 months from now. The current month counts
	// as paid, so 3 installments remain.
	startDate := monthsAgo(6)
	endDate := monthsAgo(0).AddDate(0, 3, 0)
	desc := "Parcela TV"

	original := &model.Transaction{
		ID:          1,
		IsRecurring: true,
		Amount:      500.00,
		Type:        "expense",
		Description: &desc,
		StartDate:   &startDate,
		EndDate:     &endDate,
		CreatedById: 1,
	}
	repo.transactions[1] = original

	result, err := svc.PrepayTransaction(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.RemainingMonths != 3 {
		t.Errorf("expected 3 remaining months, got %d", result.RemainingMonths)
	}
	if result.PrepaidAmount != 1500 {
		t.Errorf("expected prepaid amount 1500, got %v", result.PrepaidAmount)
	}

	// Consumption basis (F9): the original schedule stays untouched.
	if !original.EndDate.Equal(endDate) {
		t.Errorf("original end_date must not change, got %v", original.EndDate)
	}

	if result.PrepayTransaction.IsRecurring {
		t.Error("prepay transaction should not be recurring")
	}
	if result.PrepayTransaction.PrepaidFromID == nil || *result.PrepayTransaction.PrepaidFromID != 1 {
		t.Error("prepay transaction should reference original transaction")
	}
	if result.PrepayTransaction.Date == nil {
		t.Error("prepay transaction should be dated")
	}
}

func TestPrepayTransaction_FutureStartCountsAllInstallments(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	// Starts in 3 months, ends in 5 months: 3 installments, none paid (F3).
	startDate := monthsAgo(0).AddDate(0, 3, 0)
	endDate := monthsAgo(0).AddDate(0, 5, 0)

	original := &model.Transaction{
		ID:          1,
		IsRecurring: true,
		Amount:      500.00,
		Type:        "expense",
		StartDate:   &startDate,
		EndDate:     &endDate,
	}
	repo.transactions[1] = original

	result, err := svc.PrepayTransaction(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.RemainingMonths != 3 {
		t.Errorf("expected 3 remaining months for a 3-installment future plan, got %d", result.RemainingMonths)
	}
	if result.PrepaidAmount != 1500 {
		t.Errorf("expected prepaid amount 1500, got %v", result.PrepaidAmount)
	}
	if !original.EndDate.Equal(endDate) {
		t.Errorf("original end_date must not change, got %v", original.EndDate)
	}
}

func TestPrepayTransaction_NotRecurring(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	original := &model.Transaction{
		ID:          1,
		IsRecurring: false,
		Amount:      500.00,
		Type:        "expense",
	}
	repo.transactions[1] = original

	_, err := svc.PrepayTransaction(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error for non-recurring transaction")
	}

	if err.Error() != "transaction is not recurring" {
		t.Errorf("expected 'transaction is not recurring' error, got %v", err)
	}
}

func TestPrepayTransaction_NoEndDate(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	startDate := monthsAgo(3)

	original := &model.Transaction{
		ID:          1,
		IsRecurring: true,
		Amount:      500.00,
		Type:        "expense",
		StartDate:   &startDate,
		EndDate:     nil,
	}
	repo.transactions[1] = original

	_, err := svc.PrepayTransaction(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error for missing end_date")
	}

	if err.Error() != "recurring transaction has no end_date defined" {
		t.Errorf("expected 'recurring transaction has no end_date defined' error, got %v", err)
	}
}

func TestPrepayTransaction_NoStartDate(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	endDate := monthsAgo(0).AddDate(0, 3, 0)

	original := &model.Transaction{
		ID:          1,
		IsRecurring: true,
		Amount:      500.00,
		Type:        "expense",
		StartDate:   nil,
		EndDate:     &endDate,
	}
	repo.transactions[1] = original

	_, err := svc.PrepayTransaction(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error for missing start_date")
	}

	if err.Error() != "recurring transaction has no start_date defined" {
		t.Errorf("expected 'recurring transaction has no start_date defined' error, got %v", err)
	}
}

func TestPrepayTransaction_AlreadyEnded(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	startDate := monthsAgo(8)
	endDate := monthsAgo(2)

	original := &model.Transaction{
		ID:          1,
		IsRecurring: true,
		Amount:      500.00,
		Type:        "expense",
		StartDate:   &startDate,
		EndDate:     &endDate,
	}
	repo.transactions[1] = original

	_, err := svc.PrepayTransaction(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error for already ended transaction")
	}

	if err.Error() != "recurring transaction has already ended" {
		t.Errorf("expected 'recurring transaction has already ended' error, got %v", err)
	}
}

func TestPrepayTransaction_EndsThisMonthHasNothingLeft(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	startDate := monthsAgo(5)
	endDate := monthsAgo(0)

	original := &model.Transaction{
		ID:          1,
		IsRecurring: true,
		Amount:      500.00,
		Type:        "expense",
		StartDate:   &startDate,
		EndDate:     &endDate,
	}
	repo.transactions[1] = original

	_, err := svc.PrepayTransaction(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error when the current month is the last installment")
	}

	if err.Error() != "no remaining installments to prepay" {
		t.Errorf("expected 'no remaining installments to prepay' error, got %v", err)
	}
}

func TestPrepayTransaction_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	_, err := svc.PrepayTransaction(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for non-existent transaction")
	}

	if err.Error() != "record not found" {
		t.Errorf("expected 'record not found' error, got %v", err)
	}
}

func TestPrepayTransaction_DescriptionFormat(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	startDate := monthsAgo(4)
	endDate := monthsAgo(0).AddDate(0, 4, 0)
	desc := "Parcela Geladeira"

	original := &model.Transaction{
		ID:          1,
		IsRecurring: true,
		Amount:      300.00,
		Type:        "expense",
		Description: &desc,
		StartDate:   &startDate,
		EndDate:     &endDate,
		CreatedById: 1,
	}
	repo.transactions[1] = original

	result, err := svc.PrepayTransaction(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.PrepayTransaction.Description == nil {
		t.Fatal("prepay transaction should have description")
	}

	if !strings.HasPrefix(*result.PrepayTransaction.Description, "Adiantamento de") {
		t.Errorf("description should start with 'Adiantamento de', got %q", *result.PrepayTransaction.Description)
	}
}

// ---- GetAverageByCategory tests ----

func TestGetAverageByCategory_IncludesRecurring(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	catID := uint(5)
	startDate := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC) // 6 months

	recurringStart := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	// Two recurring transactions of R$500 each in "Eletrodomestics"
	repo.recurringExpenses = []repository.RecurringCategoryExpense{
		{CategoryID: &catID, CategoryName: "Eletrodomestics", Amount: 500.0, StartDate: &recurringStart, EndDate: nil},
		{CategoryID: &catID, CategoryName: "Eletrodomestics", Amount: 500.0, StartDate: &recurringStart, EndDate: nil},
	}

	result, _, err := svc.GetAverageByCategory(context.Background(), &startDate, &endDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 category, got %d", len(result))
	}

	entry := result[0]
	if entry.CategoryID != catID {
		t.Errorf("expected category ID %d, got %d", catID, entry.CategoryID)
	}

	// 2 transactions × R$500 × 6 months = R$6,000 total
	expectedTotal := 6000.0
	if entry.TotalSpent != expectedTotal {
		t.Errorf("expected TotalSpent %.2f, got %.2f", expectedTotal, entry.TotalSpent)
	}

	// R$6,000 / 6 months = R$1,000 average
	expectedAverage := 1000.0
	if entry.Average != expectedAverage {
		t.Errorf("expected Average %.2f, got %.2f", expectedAverage, entry.Average)
	}

	// No income in range → no percent_of_income
	if entry.PercentOfIncome != nil {
		t.Errorf("expected nil PercentOfIncome without income, got %v", *entry.PercentOfIncome)
	}
}

func TestGetAverageByCategory_MixesRecurringAndNonRecurring(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	catID := uint(5)
	startDate := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC) // 6 months

	recurringStart := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)

	// Non-recurring: R$300 one-off in this category
	repo.nonRecurringSums = []repository.CategoryExpenseSummary{
		{CategoryID: &catID, CategoryName: "Eletrodomestics", TotalSpent: 300.0},
	}
	// Recurring: R$500/month for 6 months = R$3,000
	repo.recurringExpenses = []repository.RecurringCategoryExpense{
		{CategoryID: &catID, CategoryName: "Eletrodomestics", Amount: 500.0, StartDate: &recurringStart, EndDate: nil},
	}

	result, _, err := svc.GetAverageByCategory(context.Background(), &startDate, &endDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 category, got %d", len(result))
	}

	// R$300 + R$3,000 = R$3,300 total; average = R$3,300 / 6 = R$550
	expectedTotal := 3300.0
	if result[0].TotalSpent != expectedTotal {
		t.Errorf("expected TotalSpent %.2f, got %.2f", expectedTotal, result[0].TotalSpent)
	}

	expectedAverage := 550.0
	if result[0].Average != expectedAverage {
		t.Errorf("expected Average %.2f, got %.2f", expectedAverage, result[0].Average)
	}
}

func TestGetAverageByCategory_PercentOfIncome(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	catID := uint(5)
	startDate := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC) // 6 months

	repo.nonRecurringSums = []repository.CategoryExpenseSummary{
		{CategoryID: &catID, CategoryName: "Housing", TotalSpent: 3000.0},
	}
	// Income: R$1,000 one-off + R$1,500/month recurring salary over all 6
	// months = R$10,000 total.
	repo.incomeTotal = 1000.0
	salaryStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	repo.recurringIncomes = []repository.RecurringAmount{
		{Amount: 1500.0, StartDate: &salaryStart, EndDate: nil},
	}

	result, totalIncome, err := svc.GetAverageByCategory(context.Background(), &startDate, &endDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if totalIncome != 10000.0 {
		t.Errorf("expected total income 10000, got %v", totalIncome)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 category, got %d", len(result))
	}
	if result[0].PercentOfIncome == nil {
		t.Fatal("expected percent_of_income to be set")
	}
	if math.Abs(*result[0].PercentOfIncome-30.0) > 0.001 {
		t.Errorf("expected percent_of_income 30, got %v", *result[0].PercentOfIncome)
	}
}

func TestGetAverageByCategory_RecurringEndedBeforeRange(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	catID := uint(5)
	startDate := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	recurringStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	recurringEnd := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC) // ended before range

	repo.recurringExpenses = []repository.RecurringCategoryExpense{
		{CategoryID: &catID, CategoryName: "Eletrodomestics", Amount: 500.0, StartDate: &recurringStart, EndDate: &recurringEnd},
	}

	result, _, err := svc.GetAverageByCategory(context.Background(), &startDate, &endDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Transaction ended before range, should contribute nothing
	if len(result) != 0 {
		t.Errorf("expected 0 categories, got %d (transaction ended before range)", len(result))
	}
}

func TestGetAverageByCategory_DefaultRangeExcludesCurrentMonth(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	catID := uint(5)
	// Recurring R$100/month, active from 4 months ago with no end. Without an
	// explicit end date the range must stop at the last complete month, so
	// the current month contributes neither spend nor a denominator slot.
	recurringStart := monthsAgo(4)
	repo.recurringExpenses = []repository.RecurringCategoryExpense{
		{CategoryID: &catID, CategoryName: "Streaming", Amount: 100.0, StartDate: &recurringStart, EndDate: nil},
	}
	earliest := recurringStart
	repo.earliestDate = &earliest

	result, _, err := svc.GetAverageByCategory(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 category, got %d", len(result))
	}

	// 4 complete months (monthsAgo(4) .. monthsAgo(1)) × R$100 / 4 = R$100.
	if result[0].TotalSpent != 400 {
		t.Errorf("expected TotalSpent 400, got %v", result[0].TotalSpent)
	}
	if result[0].Average != 100 {
		t.Errorf("expected Average 100, got %v", result[0].Average)
	}
}
