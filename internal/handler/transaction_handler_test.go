package handler

import (
	"testing"
	"time"

	"github.com/ArthurWerle/transactions/internal/model"
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
			paid := computeTotalPaid(tt.tx, now)
			left := computeTotalLeft(tt.tx, now)
			assertFloatPtr(t, "total_paid", paid, tt.wantPaid)
			assertFloatPtr(t, "total_left", left, tt.wantLeft)

			// paid + left must cover the whole schedule when both are known.
			if paid != nil && left != nil && tt.tx.EndDate != nil && tt.tx.StartDate != nil {
				totalMonths := (tt.tx.EndDate.Year()-tt.tx.StartDate.Year())*12 +
					int(tt.tx.EndDate.Month()) - int(tt.tx.StartDate.Month()) + 1
				expectedTotal := tt.tx.Amount * float64(totalMonths)
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
