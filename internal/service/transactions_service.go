package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/ArthurWerle/transactions/internal/model"
	"github.com/ArthurWerle/transactions/internal/repository"
)

// MonthlyFrequency is the only supported recurrence frequency.
const MonthlyFrequency = "monthly"

// DefaultAverageWindowMonths is the trailing window used for monthly
// averages when the caller doesn't specify one. All-time averages converge
// to a flat line and stop reflecting current behavior.
const DefaultAverageWindowMonths = 6

// ErrInvalidTransaction is the sentinel wrapped by every transaction-shape
// validation error, so handlers can map them to 400 with errors.Is.
var ErrInvalidTransaction = errors.New("invalid transaction")

func invalidf(format string, args ...interface{}) error {
	return fmt.Errorf("%w: %s", ErrInvalidTransaction, fmt.Sprintf(format, args...))
}

type PrepayResult struct {
	OriginalTransaction *model.Transaction `json:"original_transaction"`
	PrepayTransaction   *model.Transaction `json:"prepay_transaction"`
	RemainingMonths     int                `json:"remaining_months"`
	PrepaidAmount       float64            `json:"prepaid_amount"`
}

type TransactionPercentages struct {
	CategoryMonthPercent *float64 `json:"category_month_percent,omitempty"`
	TotalMonthPercent    *float64 `json:"total_month_percent,omitempty"`
}

type TransactionsService interface {
	CreateTransaction(ctx context.Context, transaction *model.Transaction) error
	GetAverageByType(ctx context.Context, windowMonths int) ([]AverageType, error)
	GetAverageByCategory(ctx context.Context, startDate, endDate *time.Time) ([]AverageByCategory, float64, error)
	GetCurrentMonthProjection(ctx context.Context) (*MonthProjection, error)
	GetTransactionByID(ctx context.Context, id uint) (*model.Transaction, error)
	GetTransactions(ctx context.Context, limit, offset int) ([]model.Transaction, int64, error)
	GetTransactionsWithFilters(ctx context.Context, currentMonth bool, categoryIDs []uint, searchQuery string, startDate, endDate *time.Time, transactionType string, limit, offset int) ([]model.Transaction, int64, error)
	GetLatestTransactions(ctx context.Context) ([]model.Transaction, error)
	GetBiggestTransactions(ctx context.Context, month, year int) ([]model.Transaction, error)
	UpdateTransaction(ctx context.Context, transaction *model.Transaction) error
	DeleteTransaction(ctx context.Context, id uint) error
	GetTransactionsByDateRange(ctx context.Context, startDate, endDate time.Time) ([]model.Transaction, error)
	GetTransactionsByCategories(ctx context.Context, categoriesIDs []uint, limit, offset int) ([]model.Transaction, error)
	PrepayTransaction(ctx context.Context, id uint) (*PrepayResult, error)
	GetTransactionMonthlyPercentages(ctx context.Context, tx *model.Transaction) (*TransactionPercentages, error)
}

type transactionsService struct {
	transactionRepo repository.TransactionsRepository
	loc             *time.Location
}

func NewTransactionsService(transactionRepo repository.TransactionsRepository, loc *time.Location) TransactionsService {
	return &transactionsService{
		transactionRepo: transactionRepo,
		loc:             loc,
	}
}

// validateTransactionShape enforces the recurring-transaction invariant:
// recurring rows are schedules (start_date set, no date, monthly frequency),
// one-offs are dated rows with no schedule fields.
func validateTransactionShape(t *model.Transaction) error {
	if t.CategoryID == 0 {
		return invalidf("category_id is required")
	}
	if t.IsRecurring {
		if t.StartDate == nil {
			return invalidf("recurring transaction requires start_date")
		}
		if t.Date != nil {
			return invalidf("recurring transaction must not have date")
		}
		if t.Frequency == nil || *t.Frequency != MonthlyFrequency {
			return invalidf("frequency must be '%s'", MonthlyFrequency)
		}
		if t.EndDate != nil && t.EndDate.Before(*t.StartDate) {
			return invalidf("end_date must not be before start_date")
		}
	} else {
		if t.Date == nil {
			return invalidf("non-recurring transaction requires date")
		}
		if t.StartDate != nil || t.EndDate != nil || t.Frequency != nil {
			return invalidf("non-recurring transaction must not have start_date, end_date or frequency")
		}
	}
	return nil
}

// normalizeTransactionShape clears fields that don't belong to the
// transaction's shape, so updates that flip is_recurring (or edit legacy rows
// created before the invariant was enforced) heal instead of erroring.
func normalizeTransactionShape(t *model.Transaction) {
	if t.IsRecurring {
		t.Date = nil
		f := MonthlyFrequency
		t.Frequency = &f
	} else {
		t.StartDate = nil
		t.EndDate = nil
		t.Frequency = nil
	}
}

func (s *transactionsService) CreateTransaction(ctx context.Context, transaction *model.Transaction) error {
	if transaction.IsRecurring && transaction.Frequency == nil {
		f := MonthlyFrequency
		transaction.Frequency = &f
	}
	if err := validateTransactionShape(transaction); err != nil {
		return err
	}
	return s.transactionRepo.Create(transaction)
}

func (s *transactionsService) GetTransactionByID(ctx context.Context, id uint) (*model.Transaction, error) {
	return s.transactionRepo.FindByID(id)
}

func (s *transactionsService) GetTransactions(ctx context.Context, limit, offset int) ([]model.Transaction, int64, error) {
	total, err := s.transactionRepo.CountAll()
	if err != nil {
		return nil, 0, err
	}
	transactions, err := s.transactionRepo.FindAll(limit, offset)
	return transactions, total, err
}

func (s *transactionsService) GetLatestTransactions(ctx context.Context) ([]model.Transaction, error) {
	return s.transactionRepo.FindLatest()
}

func (s *transactionsService) GetBiggestTransactions(ctx context.Context, month, year int) ([]model.Transaction, error) {
	return s.transactionRepo.FindBiggest(month, year)
}

func (s *transactionsService) GetTransactionsWithFilters(ctx context.Context, currentMonth bool, categoryIDs []uint, searchQuery string, startDate, endDate *time.Time, transactionType string, limit, offset int) ([]model.Transaction, int64, error) {
	total, err := s.transactionRepo.CountAllWithFilters(currentMonth, categoryIDs, searchQuery, startDate, endDate, transactionType)
	if err != nil {
		return nil, 0, err
	}
	transactions, err := s.transactionRepo.FindAllWithFilters(currentMonth, categoryIDs, searchQuery, startDate, endDate, transactionType, limit, offset)
	return transactions, total, err
}

func (s *transactionsService) UpdateTransaction(ctx context.Context, transaction *model.Transaction) error {
	normalizeTransactionShape(transaction)
	if err := validateTransactionShape(transaction); err != nil {
		return err
	}
	return s.transactionRepo.Update(transaction)
}

func (s *transactionsService) DeleteTransaction(ctx context.Context, id uint) error {
	return s.transactionRepo.Delete(id)
}

type AverageType struct {
	TypeName string
	Average  float64
}

type AverageByCategory struct {
	CategoryID      uint     `json:"category_id"`
	CategoryName    string   `json:"category_name"`
	Average         float64  `json:"average"`
	TotalSpent      float64  `json:"total_spent"`
	PercentOfIncome *float64 `json:"percent_of_income,omitempty"`
}

// MonthStart normalizes a time to the first instant of its calendar month. For
// "now" values, convert to the reporting location before calling.
func MonthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// GetAverageByType computes each type's monthly average over that type's own
// active range, using only complete months (the in-progress month would
// dilute the average). Types whose data lives entirely in the current month
// fall back to reporting the current month's total. Averages cover a
// trailing window of the last windowMonths complete months (M1), so the
// number tracks recent behavior instead of converging to an all-time flat
// line.
func (s *transactionsService) GetAverageByType(ctx context.Context, windowMonths int) ([]AverageType, error) {
	if windowMonths < 1 {
		windowMonths = DefaultAverageWindowMonths
	}

	monthlyTotals, err := s.transactionRepo.FindNonRecurringMonthlyTotalsByType()
	if err != nil {
		log.Printf("[TransactionsService.GetAverageByType] ERROR: Failed to fetch monthly totals: %v", err)
		return nil, fmt.Errorf("failed to fetch monthly totals by type: %w", err)
	}

	recurringTxs, err := s.transactionRepo.FindRecurringTransactionSummaryByType()
	if err != nil {
		log.Printf("[TransactionsService.GetAverageByType] ERROR: Failed to fetch recurring transactions: %v", err)
		return nil, fmt.Errorf("failed to fetch recurring transactions by type: %w", err)
	}

	now := time.Now().In(s.loc)
	currentMonth := MonthStart(now)
	lastComplete := currentMonth.AddDate(0, -1, 0)
	windowStart := lastComplete.AddDate(0, -(windowMonths-1), 0)

	type agg struct {
		total    float64
		earliest time.Time
		set      bool
	}
	completed := make(map[string]*agg)
	currentOnly := make(map[string]float64)

	add := func(typeName string, amount float64, month time.Time) {
		a := completed[typeName]
		if a == nil {
			a = &agg{}
			completed[typeName] = a
		}
		a.total += amount
		if !a.set || month.Before(a.earliest) {
			a.earliest = month
			a.set = true
		}
	}

	for _, row := range monthlyTotals {
		m := time.Date(row.Year, time.Month(row.Month), 1, 0, 0, 0, 0, time.UTC)
		switch {
		case m.Before(windowStart):
			// outside the trailing window
		case m.Before(currentMonth):
			add(row.Type, row.MonthlySum, m)
		case m.Equal(currentMonth):
			currentOnly[row.Type] += row.MonthlySum
		}
	}

	for _, tx := range recurringTxs {
		if tx.StartDate == nil {
			continue
		}
		startM := MonthStart(*tx.StartDate)
		if startM.Before(windowStart) {
			startM = windowStart
		}
		endM := lastComplete
		if tx.EndDate != nil {
			if e := MonthStart(*tx.EndDate); e.Before(endM) {
				endM = e
			}
		}
		if !endM.Before(startM) {
			add(tx.Type, tx.Amount*float64(InclusiveMonthCount(startM, endM)), startM)
		}
		startedByNow := !MonthStart(*tx.StartDate).After(currentMonth)
		activeNow := startedByNow && (tx.EndDate == nil || !MonthStart(*tx.EndDate).Before(currentMonth))
		if activeNow {
			currentOnly[tx.Type] += tx.Amount
		}
	}

	var result []AverageType
	for typeName, a := range completed {
		den := InclusiveMonthCount(a.earliest, lastComplete)
		if den < 1 {
			den = 1
		}
		result = append(result, AverageType{
			TypeName: typeName,
			Average:  a.total / float64(den),
		})
	}
	for typeName, total := range currentOnly {
		if _, ok := completed[typeName]; !ok {
			result = append(result, AverageType{TypeName: typeName, Average: total})
		}
	}

	return result, nil
}

// GetAverageByCategory returns per-category monthly expense averages plus the
// total income over the same range (for percent-of-income). When no end date
// is supplied the range stops at the last complete month.
func (s *transactionsService) GetAverageByCategory(ctx context.Context, startDate, endDate *time.Time) ([]AverageByCategory, float64, error) {
	now := time.Now().In(s.loc)
	currentMonth := MonthStart(now)
	lastComplete := currentMonth.AddDate(0, -1, 0)

	lastDayOf := func(month time.Time) time.Time {
		return month.AddDate(0, 1, -1)
	}

	effectiveEnd := endDate
	if effectiveEnd == nil {
		e := lastDayOf(lastComplete)
		effectiveEnd = &e
	}

	earliestDate, err := s.transactionRepo.FindEarliestDate(startDate, effectiveEnd)
	if err != nil {
		log.Printf("[TransactionsService.GetAverageByCategory] ERROR: Failed to fetch earliest date: %v", err)
		return nil, 0, fmt.Errorf("failed to fetch earliest transaction date: %w", err)
	}

	var rangeStart time.Time
	if startDate != nil {
		rangeStart = MonthStart(*startDate)
	} else if earliestDate != nil {
		rangeStart = MonthStart(earliestDate.In(s.loc))
	} else {
		rangeStart = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	rangeEnd := MonthStart(*effectiveEnd)

	// All data in the current (partial) month: include it rather than
	// reporting an empty range.
	if endDate == nil && rangeEnd.Before(rangeStart) {
		rangeEnd = currentMonth
		e := lastDayOf(currentMonth)
		effectiveEnd = &e
	}

	totalMonths := InclusiveMonthCount(rangeStart, rangeEnd)
	if totalMonths < 1 {
		totalMonths = 1
	}

	summaries, err := s.transactionRepo.FindExpenseSummaryByCategory(startDate, effectiveEnd)
	if err != nil {
		log.Printf("[TransactionsService.GetAverageByCategory] ERROR: Failed to fetch summaries: %v", err)
		return nil, 0, fmt.Errorf("failed to fetch expense summaries by category: %w", err)
	}

	type categoryInfo struct {
		name  string
		total float64
	}
	categoryMap := make(map[uint]*categoryInfo)
	for _, summary := range summaries {
		catID := uint(0)
		if summary.CategoryID != nil {
			catID = *summary.CategoryID
		}
		categoryMap[catID] = &categoryInfo{
			name:  summary.CategoryName,
			total: summary.TotalSpent,
		}
	}

	recurringExpenses, err := s.transactionRepo.FindRecurringExpensesInRange(startDate, effectiveEnd)
	if err != nil {
		log.Printf("[TransactionsService.GetAverageByCategory] ERROR: Failed to fetch recurring expenses: %v", err)
		return nil, 0, fmt.Errorf("failed to fetch recurring expense summaries by category: %w", err)
	}

	activeMonths := func(startD *time.Time, endD *time.Time) int {
		activeStart := rangeStart
		if s := MonthStart(*startD); s.After(activeStart) {
			activeStart = s
		}
		activeEnd := rangeEnd
		if endD != nil {
			if e := MonthStart(*endD); e.Before(activeEnd) {
				activeEnd = e
			}
		}
		return InclusiveMonthCount(activeStart, activeEnd)
	}

	for _, tx := range recurringExpenses {
		if tx.CategoryID == nil || tx.StartDate == nil {
			continue
		}

		months := activeMonths(tx.StartDate, tx.EndDate)
		if months < 1 {
			continue
		}

		catID := *tx.CategoryID
		contribution := tx.Amount * float64(months)
		if info, exists := categoryMap[catID]; exists {
			info.total += contribution
		} else {
			categoryMap[catID] = &categoryInfo{
				name:  tx.CategoryName,
				total: contribution,
			}
		}
	}

	incomeTotal, err := s.transactionRepo.FindIncomeTotalInRange(startDate, effectiveEnd)
	if err != nil {
		log.Printf("[TransactionsService.GetAverageByCategory] ERROR: Failed to fetch income total: %v", err)
		return nil, 0, fmt.Errorf("failed to fetch income total: %w", err)
	}
	recurringIncomes, err := s.transactionRepo.FindRecurringIncomesInRange(startDate, effectiveEnd)
	if err != nil {
		log.Printf("[TransactionsService.GetAverageByCategory] ERROR: Failed to fetch recurring incomes: %v", err)
		return nil, 0, fmt.Errorf("failed to fetch recurring incomes: %w", err)
	}
	for _, tx := range recurringIncomes {
		if tx.StartDate == nil {
			continue
		}
		if months := activeMonths(tx.StartDate, tx.EndDate); months >= 1 {
			incomeTotal += tx.Amount * float64(months)
		}
	}

	var result []AverageByCategory
	for catID, info := range categoryMap {
		item := AverageByCategory{
			CategoryID:   catID,
			CategoryName: info.name,
			Average:      info.total / float64(totalMonths),
			TotalSpent:   info.total,
		}
		if incomeTotal > 0 {
			pct := info.total / incomeTotal * 100
			item.PercentOfIncome = &pct
		}
		result = append(result, item)
	}

	return result, incomeTotal, nil
}

type MonthProjection struct {
	Month              string  `json:"month"`
	RecurringCommitted float64 `json:"recurring_committed"`
	OneOffSpent        float64 `json:"one_off_spent"`
	ProjectedOneOff    float64 `json:"projected_one_off"`
	ProjectedTotal     float64 `json:"projected_total"`
}

// projectMonth extrapolates end-of-month spending: recurring commitments are
// fixed, one-off spending continues at the month-to-date daily run rate.
func projectMonth(now time.Time, recurringCommitted, oneOffSpent float64) *MonthProjection {
	daysInMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day()
	elapsedDays := now.Day()

	projectedOneOff := oneOffSpent / float64(elapsedDays) * float64(daysInMonth)

	return &MonthProjection{
		Month:              now.Format("2006-01"),
		RecurringCommitted: recurringCommitted,
		OneOffSpent:        oneOffSpent,
		ProjectedOneOff:    projectedOneOff,
		ProjectedTotal:     recurringCommitted + projectedOneOff,
	}
}

// GetCurrentMonthProjection estimates where this month's expenses will land
// (M6): the recurring commitments already known plus one-off spending
// extrapolated from the month-to-date run rate.
func (s *transactionsService) GetCurrentMonthProjection(ctx context.Context) (*MonthProjection, error) {
	recurring, err := s.transactionRepo.FindCurrentMonthRecurringExpenseTotal()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recurring expense total: %w", err)
	}
	oneOff, err := s.transactionRepo.FindMonthToDateOneOffExpenseTotal()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch month-to-date expense total: %w", err)
	}
	return projectMonth(time.Now().In(s.loc), recurring, oneOff), nil
}

func (s *transactionsService) GetTransactionsByDateRange(ctx context.Context, startDate, endDate time.Time) ([]model.Transaction, error) {
	return s.transactionRepo.FindByDateRange(startDate, endDate)
}

func (s *transactionsService) GetTransactionsByCategories(ctx context.Context, categoriesIDs []uint, limit, offset int) ([]model.Transaction, error) {
	return s.transactionRepo.FindByCategories(categoriesIDs, limit, offset)
}

// PrepayTransaction records the remaining installments of a recurring
// transaction as a single dated payment. The original schedule is left
// untouched: aggregates keep attributing the monthly amount to each month it
// covers (consumption basis), while the prepayment row exists as the cash
// record and is excluded from aggregates via prepaid_from_id.
func (s *transactionsService) PrepayTransaction(ctx context.Context, id uint) (*PrepayResult, error) {
	original, err := s.transactionRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	if original.IsPrepaid {
		return nil, errors.New("transaction was already prepaid")
	}

	if !original.IsRecurring {
		return nil, errors.New("transaction is not recurring")
	}

	if original.EndDate == nil {
		return nil, errors.New("recurring transaction has no end_date defined")
	}

	if original.StartDate == nil {
		return nil, errors.New("recurring transaction has no start_date defined")
	}

	now := time.Now().In(s.loc)
	currentMonth := MonthStart(now)
	startMonth := MonthStart(*original.StartDate)
	endMonth := MonthStart(*original.EndDate)

	if endMonth.Before(currentMonth) {
		return nil, errors.New("recurring transaction has already ended")
	}

	var remainingMonths int
	if startMonth.After(currentMonth) {
		// The schedule hasn't started yet: every installment is still owed.
		remainingMonths = InclusiveMonthCount(startMonth, endMonth)
	} else {
		// The current month's installment counts as already paid.
		remainingMonths = ExclusiveMonthCount(currentMonth, endMonth)
	}
	if remainingMonths <= 0 {
		return nil, errors.New("no remaining installments to prepay")
	}

	prepaidAmount := original.Amount * float64(remainingMonths)

	originalDesc := ""
	if original.Description != nil {
		originalDesc = *original.Description
	}
	prepayDesc := fmt.Sprintf("Adiantamento de %d parcelas de %s", remainingMonths, originalDesc)

	txDate := time.Now()
	prepayment := &model.Transaction{
		IsRecurring:   false,
		CategoryID:    original.CategoryID,
		CreatedById:   original.CreatedById,
		Amount:        prepaidAmount,
		Type:          original.Type,
		Subtype:       original.Subtype,
		Description:   &prepayDesc,
		Date:          &txDate,
		PrepaidFromID: &original.ID,
	}

	if err := s.transactionRepo.Create(prepayment); err != nil {
		return nil, err
	}

	return &PrepayResult{
		OriginalTransaction: original,
		PrepayTransaction:   prepayment,
		RemainingMonths:     remainingMonths,
		PrepaidAmount:       prepaidAmount,
	}, nil
}

// InclusiveMonthCount returns the number of calendar months spanned, counting both endpoints.
// e.g. Jan → Jan = 1, Jan → Mar = 3.
func InclusiveMonthCount(start, end time.Time) int {
	years := end.Year() - start.Year()
	months := int(end.Month()) - int(start.Month())
	return years*12 + months + 1
}

// ExclusiveMonthCount returns the number of months strictly after start and up to end.
// Used when the current month is already paid and only future months remain.
func ExclusiveMonthCount(start, end time.Time) int {
	return InclusiveMonthCount(start, end) - 1
}

func (s *transactionsService) GetTransactionMonthlyPercentages(ctx context.Context, tx *model.Transaction) (*TransactionPercentages, error) {
	monthTotal, err := s.transactionRepo.FindCurrentMonthTotalByType(tx.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch current month total: %w", err)
	}

	percentages := &TransactionPercentages{}

	if monthTotal > 0 {
		pct := tx.Amount / monthTotal * 100
		percentages.TotalMonthPercent = &pct
	}

	if tx.CategoryID != 0 {
		categoryTotal, err := s.transactionRepo.FindCurrentMonthTotalByTypeAndCategory(tx.Type, tx.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch current month category total: %w", err)
		}
		if categoryTotal > 0 {
			pct := tx.Amount / categoryTotal * 100
			percentages.CategoryMonthPercent = &pct
		}
	}

	return percentages, nil
}
