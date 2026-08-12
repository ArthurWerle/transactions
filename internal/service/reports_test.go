package service

import (
	"context"
	"testing"
	"time"

	"github.com/ArthurWerle/transactions/internal/model"
	"github.com/ArthurWerle/transactions/internal/repository"
)

func datePtr(year int, month time.Month, day int) *time.Time {
	d := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &d
}

func recurring(start, end *time.Time, amount float64) *model.Transaction {
	return &model.Transaction{IsRecurring: true, StartDate: start, EndDate: end, Amount: amount}
}

// The convention: the current month counts as paid, so paid + left always
// covers the whole schedule (F4).
func TestComputeTotalPaidAndLeft(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		tx       *model.Transaction
		wantPaid *float64
		wantLeft *float64
	}{
		{
			name:     "mid-schedule: Jan-Dec 2026 viewed in July",
			tx:       recurring(datePtr(2026, 1, 10), datePtr(2026, 12, 10), 100),
			wantPaid: f(700), // Jan..Jul = 7 months
			wantLeft: f(500), // Aug..Dec = 5 months
		},
		{
			name:     "starts this month",
			tx:       recurring(datePtr(2026, 7, 1), datePtr(2026, 9, 1), 100),
			wantPaid: f(100),
			wantLeft: f(200),
		},
		{
			name:     "ends this month",
			tx:       recurring(datePtr(2026, 5, 1), datePtr(2026, 7, 1), 100),
			wantPaid: f(300),
			wantLeft: f(0),
		},
		{
			name:     "already ended",
			tx:       recurring(datePtr(2026, 1, 1), datePtr(2026, 3, 1), 100),
			wantPaid: f(300),
			wantLeft: f(0),
		},
		{
			name:     "starts in the future",
			tx:       recurring(datePtr(2026, 10, 1), datePtr(2026, 12, 1), 100),
			wantPaid: f(0),
			wantLeft: f(300),
		},
		{
			name:     "no end date: paid accrues, left unknown",
			tx:       recurring(datePtr(2026, 4, 1), nil, 100),
			wantPaid: f(400), // Apr..Jul
			wantLeft: nil,
		},
		{
			name:     "non-recurring: neither applies",
			tx:       &model.Transaction{Amount: 100},
			wantPaid: nil,
			wantLeft: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paid := ComputeTotalPaid(tt.tx, now)
			left := ComputeTotalLeft(tt.tx, now)
			assertFloatPtr(t, "total_paid", paid, tt.wantPaid)
			assertFloatPtr(t, "total_left", left, tt.wantLeft)

			if paid != nil && left != nil && tt.tx.EndDate != nil && tt.tx.StartDate != nil {
				expectedTotal := tt.tx.Amount * float64(InclusiveMonthCount(MonthStart(*tt.tx.StartDate), MonthStart(*tt.tx.EndDate)))
				if *paid+*left != expectedTotal {
					t.Errorf("paid (%v) + left (%v) = %v, expected schedule total %v", *paid, *left, *paid+*left, expectedTotal)
				}
			}
		})
	}
}

func f(v float64) *float64 { return &v }

func assertFloatPtr(t *testing.T, label string, got, want *float64) {
	t.Helper()
	switch {
	case want == nil && got != nil:
		t.Errorf("%s: expected nil, got %v", label, *got)
	case want != nil && got == nil:
		t.Errorf("%s: expected %v, got nil", label, *want)
	case want != nil && got != nil && *want != *got:
		t.Errorf("%s: expected %v, got %v", label, *want, *got)
	}
}

// ---- report shaping tests (the SQL itself is covered by the repository
// integration tests) ----

func month(y int, m time.Month) time.Time {
	return time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
}

func TestGetMonthlyHistory_LabelsAndBalance(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	repo.monthlyFlow = []repository.MonthlyFlowRow{
		{Month: month(2026, 5), Income: 5000, Expense: 3200},
		{Month: month(2026, 6), Income: 0, Expense: 0},
	}

	points, err := svc.GetMonthlyHistory(context.Background(), month(2026, 5), month(2026, 6))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(points))
	}
	if points[0].Month != "May 26" || points[1].Month != "Jun 26" {
		t.Errorf("unexpected month labels: %q, %q", points[0].Month, points[1].Month)
	}
	if points[0].Balance != 1800 {
		t.Errorf("expected balance 1800, got %v", points[0].Balance)
	}
	if points[1].Income != 0 || points[1].Expense != 0 || points[1].Balance != 0 {
		t.Errorf("expected zero-filled month, got %+v", points[1])
	}
}

func TestGetCategoryHistory_GroupsSeriesAndSkipsEmptyMonths(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	repo.categoryFlow = []repository.CategoryMonthlyFlowRow{
		{Month: month(2026, 5), CategoryID: 1, CategoryName: "Food", Color: "#111", Expense: 200},
		{Month: month(2026, 6), CategoryID: 1, CategoryName: "Food", Color: "#111", Expense: 250},
		{Month: month(2026, 6), CategoryID: 2, CategoryName: "Old Hobby (deleted)", Color: "#222", Expense: 40},
		{Month: month(2026, 7), CategoryID: 2, CategoryName: "Old Hobby (deleted)", Color: "#222"},
	}

	series, err := svc.GetCategoryHistory(context.Background(), month(2026, 5), month(2026, 7), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(series) != 2 {
		t.Fatalf("expected 2 series, got %d", len(series))
	}
	if series[0].CategoryName != "Food" || len(series[0].Data) != 2 {
		t.Errorf("unexpected first series: %+v", series[0])
	}
	// The all-zero July row must be dropped from the deleted category.
	if len(series[1].Data) != 1 || series[1].Data[0].Expense != 40 {
		t.Errorf("expected a single active month for the deleted category, got %+v", series[1].Data)
	}
}

func TestGetMonthOverview_Variations(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	// May: income 100, expense 0 · June: income 150, expense 300
	repo.monthlyFlow = []repository.MonthlyFlowRow{
		{Month: month(2026, 5), Income: 100, Expense: 0},
		{Month: month(2026, 6), Income: 150, Expense: 300},
	}

	overview, err := svc.GetMonthOverview(context.Background(), 6, 2026)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if overview.Income.CurrentMonth != 150 || overview.Income.LastMonth != 100 {
		t.Errorf("unexpected income overview: %+v", overview.Income)
	}
	if overview.Income.PercentageVariation == nil || *overview.Income.PercentageVariation != 50 {
		t.Errorf("expected +50%% income variation, got %v", overview.Income.PercentageVariation)
	}
	// Division by zero must yield null, not Inf (the contract charts rely on).
	if overview.Expense.PercentageVariation != nil {
		t.Errorf("expected nil expense variation for zero last month, got %v", *overview.Expense.PercentageVariation)
	}
}

func TestGetMonthlyExpensesBySubcategory(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	// Repository already returns largest-first with a "(none)" bucket for
	// transactions that have no subcategory; the service passes it through.
	repo.subcategoryMonthTotals = []repository.SubcategoryMonthTotal{
		{SubcategoryName: "Groceries", Total: 300},
		{SubcategoryName: "(none)", Total: 120},
		{SubcategoryName: "Dining (deleted)", Total: 50},
	}

	got, err := svc.GetMonthlyExpensesBySubcategory(context.Background(), 6, 2026)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(got))
	}
	if got[0].SubcategoryName != "Groceries" || got[0].Total != 300 {
		t.Errorf("unexpected first row: %+v", got[0])
	}
	if got[1].SubcategoryName != "(none)" || got[1].Total != 120 {
		t.Errorf("expected the (none) bucket preserved, got %+v", got[1])
	}
}

func TestGetMonthlyExpensesByLocation(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	repo.locationMonthTotals = []repository.LocationMonthTotal{
		{LocationName: "Supermarket", Total: 400},
		{LocationName: "(none)", Total: 75},
	}

	got, err := svc.GetMonthlyExpensesByLocation(context.Background(), 6, 2026)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(got))
	}
	if got[0].LocationName != "Supermarket" || got[0].Total != 400 {
		t.Errorf("unexpected first row: %+v", got[0])
	}
	if got[1].LocationName != "(none)" || got[1].Total != 75 {
		t.Errorf("expected the (none) bucket preserved, got %+v", got[1])
	}
}

func TestGetMonthlyDailyExpenses_FormatsDaysAndSumsTotal(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	// The repository already zero-fills every day; the service formats each
	// day's date and sums the totals (which reconcile with the monthly total).
	repo.dailyExpenseTotals = []repository.DailyExpenseTotal{
		{Day: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), Total: 10},
		{Day: time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC), Total: 0},
		{Day: time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC), Total: 25.5},
	}
	repo.dailyExpenseCount = 7

	got, err := svc.GetMonthlyDailyExpenses(context.Background(), 6, 2026)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Days) != 3 {
		t.Fatalf("expected 3 days, got %d", len(got.Days))
	}
	if got.Days[0].Date != "2026-06-01" || got.Days[0].Total != 10 {
		t.Errorf("unexpected first day: %+v", got.Days[0])
	}
	if got.TransactionCount != 7 {
		t.Errorf("expected count 7, got %d", got.TransactionCount)
	}
	if got.Total != 35.5 {
		t.Errorf("expected summed total 35.5, got %v", got.Total)
	}
}

func TestGetMonthlyMerchants_PassesThroughRows(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo)

	repo.merchantMonthTotals = []repository.MerchantMonthTotal{
		{Name: "Supermarket", Total: 400, TransactionCount: 12, TopCategory: "Groceries"},
		{Name: "(none)", Total: 75, TransactionCount: 3, TopCategory: "Misc"},
	}

	got, err := svc.GetMonthlyMerchants(context.Background(), 6, 2026)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(got))
	}
	if got[0].Name != "Supermarket" || got[0].Total != 400 || got[0].TransactionCount != 12 || got[0].TopCategory != "Groceries" {
		t.Errorf("unexpected first merchant: %+v", got[0])
	}
	if got[1].Name != "(none)" || got[1].TransactionCount != 3 {
		t.Errorf("expected the (none) bucket preserved, got %+v", got[1])
	}
}
