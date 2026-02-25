package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ArthurWerle/transactions/internal/model"
	"github.com/ArthurWerle/transactions/internal/repository"
)

type mockTransactionsRepository struct {
	transactions       map[uint]*model.Transaction
	prepayErr          error
	nonRecurringSums   []repository.CategoryExpenseSummary
	recurringExpenses  []repository.RecurringCategoryExpense
	earliestDate       *time.Time
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
	return nil
}

func (m *mockTransactionsRepository) FindByID(id uint) (*model.Transaction, error) {
	t, ok := m.transactions[id]
	if !ok {
		return nil, errors.New("record not found")
	}
	return t, nil
}

func (m *mockTransactionsRepository) FindAll() ([]model.Transaction, error) {
	return nil, nil
}

func (m *mockTransactionsRepository) FindAllWithFilters(currentMonth bool, categoryIDs []uint, searchQuery string, startDate, endDate *time.Time, transactionType string) ([]model.Transaction, error) {
	return nil, nil
}

func (m *mockTransactionsRepository) FindBiggest() ([]model.Transaction, error) {
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

func (m *mockTransactionsRepository) FindByType(transactionType string, limit, offset int) ([]model.Transaction, error) {
	return nil, nil
}

func (m *mockTransactionsRepository) FindRecurring() ([]model.Transaction, error) {
	return nil, nil
}

func (m *mockTransactionsRepository) FindByCategories(categoriesIDs []uint, limit, offset int) ([]model.Transaction, error) {
	return nil, nil
}

func (m *mockTransactionsRepository) FindByCategory(categoryID uint, limit, offset int) ([]model.Transaction, error) {
	return nil, nil
}

func (m *mockTransactionsRepository) PrepayTransaction(original *model.Transaction, prepayment *model.Transaction) error {
	if m.prepayErr != nil {
		return m.prepayErr
	}
	m.transactions[original.ID] = original
	prepayment.ID = uint(len(m.transactions) + 1)
	m.transactions[prepayment.ID] = prepayment
	return nil
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

func (m *mockTransactionsRepository) FindCurrentMonthTotalByType(transactionType string) (float64, error) {
	return 0, nil
}

func (m *mockTransactionsRepository) FindCurrentMonthTotalByTypeAndCategory(transactionType string, categoryID uint) (float64, error) {
	return 0, nil
}

func TestPrepayTransaction_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewTransactionsService(repo)

	startDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
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

	if result.PrepayTransaction == nil {
		t.Fatal("expected prepay transaction to be created")
	}

	if result.PrepayTransaction.IsRecurring {
		t.Error("prepay transaction should not be recurring")
	}

	if result.PrepayTransaction.PrepaidFromID == nil || *result.PrepayTransaction.PrepaidFromID != 1 {
		t.Error("prepay transaction should reference original transaction")
	}

	if result.RemainingMonths <= 0 {
		t.Error("should have remaining months")
	}

	expectedAmount := original.Amount * float64(result.RemainingMonths)
	if result.PrepaidAmount != expectedAmount {
		t.Errorf("expected prepaid amount %v, got %v", expectedAmount, result.PrepaidAmount)
	}
}

func TestPrepayTransaction_NotRecurring(t *testing.T) {
	repo := newMockRepository()
	svc := NewTransactionsService(repo)

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
	svc := NewTransactionsService(repo)

	startDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

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
	svc := NewTransactionsService(repo)

	endDate := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

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
	svc := NewTransactionsService(repo)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

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

func TestPrepayTransaction_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewTransactionsService(repo)

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
	svc := NewTransactionsService(repo)

	startDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
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

	expectedDescPrefix := "Adiantamento de"
	if len(*result.PrepayTransaction.Description) < len(expectedDescPrefix) {
		t.Errorf("description should start with '%s'", expectedDescPrefix)
	}
}

func TestGetAverageByCategory_IncludesRecurring(t *testing.T) {
	repo := newMockRepository()
	svc := NewTransactionsService(repo)

	catID := uint(5)
	startDate := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC) // 6 months

	recurringStart := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	// Two recurring transactions of R$500 each in "Eletrodomestics"
	repo.recurringExpenses = []repository.RecurringCategoryExpense{
		{CategoryID: &catID, CategoryName: "Eletrodomestics", Amount: 500.0, StartDate: &recurringStart, EndDate: nil},
		{CategoryID: &catID, CategoryName: "Eletrodomestics", Amount: 500.0, StartDate: &recurringStart, EndDate: nil},
	}

	result, err := svc.GetAverageByCategory(context.Background(), &startDate, &endDate)
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
}

func TestGetAverageByCategory_MixesRecurringAndNonRecurring(t *testing.T) {
	repo := newMockRepository()
	svc := NewTransactionsService(repo)

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

	result, err := svc.GetAverageByCategory(context.Background(), &startDate, &endDate)
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

func TestGetAverageByCategory_RecurringEndedBeforeRange(t *testing.T) {
	repo := newMockRepository()
	svc := NewTransactionsService(repo)

	catID := uint(5)
	startDate := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	recurringStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	recurringEnd := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC) // ended before range

	repo.recurringExpenses = []repository.RecurringCategoryExpense{
		{CategoryID: &catID, CategoryName: "Eletrodomestics", Amount: 500.0, StartDate: &recurringStart, EndDate: &recurringEnd},
	}

	result, err := svc.GetAverageByCategory(context.Background(), &startDate, &endDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Transaction ended before range, should contribute nothing
	if len(result) != 0 {
		t.Errorf("expected 0 categories, got %d (transaction ended before range)", len(result))
	}
}

func TestMonthsBetween(t *testing.T) {
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
			name:     "5 months apart",
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
			result := monthsBetween(tt.start, tt.end)
			if result != tt.expected {
				t.Errorf("monthsBetween(%v, %v) = %d, expected %d", tt.start, tt.end, result, tt.expected)
			}
		})
	}
}
