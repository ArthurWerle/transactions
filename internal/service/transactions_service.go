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
	GetAverageByType(ctx context.Context) ([]AverageType, error)
	GetAverageByCategory(ctx context.Context, startDate, endDate *time.Time) ([]AverageByCategory, error)
	GetTransactionByID(ctx context.Context, id uint) (*model.Transaction, error)
	GetTransactions(ctx context.Context, limit, offset int) ([]model.Transaction, int64, error)
	GetTransactionsWithFilters(ctx context.Context, currentMonth bool, categoryIDs []uint, searchQuery string, startDate, endDate *time.Time, transactionType string, limit, offset int) ([]model.Transaction, int64, error)
	GetLatestTransactions(ctx context.Context) ([]model.Transaction, error)
	GetBiggestTransactions(ctx context.Context, month, year int) ([]model.Transaction, error)
	UpdateTransaction(ctx context.Context, transaction *model.Transaction) error
	DeleteTransaction(ctx context.Context, id uint) error
	GetTransactionsByDateRange(ctx context.Context, startDate, endDate time.Time) ([]model.Transaction, error)
	GetTransactionsByType(ctx context.Context, transactionType string, limit, offset int) ([]model.Transaction, error)
	GetRecurringTransactions(ctx context.Context) ([]model.Transaction, error)
	GetTransactionsByCategory(ctx context.Context, categoryID uint, limit, offset int) ([]model.Transaction, error)
	GetTransactionsByCategories(ctx context.Context, categoriesIDs []uint, limit, offset int) ([]model.Transaction, error)
	PrepayTransaction(ctx context.Context, id uint) (*PrepayResult, error)
	GetTransactionMonthlyPercentages(ctx context.Context, tx *model.Transaction) (*TransactionPercentages, error)
}

type transactionsService struct {
	transactionRepo repository.TransactionsRepository
}

func NewTransactionsService(transactionRepo repository.TransactionsRepository) TransactionsService {
	return &transactionsService{
		transactionRepo: transactionRepo,
	}
}

func (s *transactionsService) CreateTransaction(ctx context.Context, transaction *model.Transaction) error {
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
	CategoryID   uint    `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Average      float64 `json:"average"`
	TotalSpent   float64 `json:"total_spent"`
}

func (s *transactionsService) GetAverageByType(ctx context.Context) ([]AverageType, error) {
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

	type yearMonth struct{ year, month int }
	typeMonthBuckets := make(map[string]map[yearMonth]float64)

	// Determine global range start: earliest across non-recurring and recurring start dates
	var rangeStartYear, rangeStartMonth int
	rangeStartSet := false

	updateRangeStart := func(y, m int) {
		if !rangeStartSet || y < rangeStartYear || (y == rangeStartYear && m < rangeStartMonth) {
			rangeStartYear = y
			rangeStartMonth = m
			rangeStartSet = true
		}
	}

	for _, row := range monthlyTotals {
		if typeMonthBuckets[row.Type] == nil {
			typeMonthBuckets[row.Type] = make(map[yearMonth]float64)
		}
		typeMonthBuckets[row.Type][yearMonth{row.Year, row.Month}] += row.MonthlySum
		updateRangeStart(row.Year, row.Month)
	}

	for _, tx := range recurringTxs {
		if tx.StartDate == nil {
			continue
		}
		updateRangeStart(tx.StartDate.Year(), int(tx.StartDate.Month()))
	}

	if !rangeStartSet {
		return nil, nil
	}

	now := time.Now()
	rangeStart := time.Date(rangeStartYear, time.Month(rangeStartMonth), 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	totalMonths := inclusiveMonthCount(rangeStart, rangeEnd)
	if totalMonths < 1 {
		totalMonths = 1
	}

	// Expand each recurring transaction into per-month buckets within its active range
	for _, tx := range recurringTxs {
		if tx.StartDate == nil {
			continue
		}
		if typeMonthBuckets[tx.Type] == nil {
			typeMonthBuckets[tx.Type] = make(map[yearMonth]float64)
		}
		txEnd := rangeEnd
		if tx.EndDate != nil {
			e := time.Date(tx.EndDate.Year(), tx.EndDate.Month(), 1, 0, 0, 0, 0, time.UTC)
			if e.Before(txEnd) {
				txEnd = e
			}
		}
		cur := time.Date(tx.StartDate.Year(), tx.StartDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		for !cur.After(txEnd) {
			typeMonthBuckets[tx.Type][yearMonth{cur.Year(), int(cur.Month())}] += tx.Amount
			cur = cur.AddDate(0, 1, 0)
		}
	}

	var result []AverageType
	for typeName, buckets := range typeMonthBuckets {
		var total float64
		for _, sum := range buckets {
			total += sum
		}
		result = append(result, AverageType{
			TypeName: typeName,
			Average:  total / float64(totalMonths),
		})
	}

	return result, nil
}

func (s *transactionsService) GetAverageByCategory(ctx context.Context, startDate, endDate *time.Time) ([]AverageByCategory, error) {
	earliestDate, err := s.transactionRepo.FindEarliestDate(startDate, endDate)
	if err != nil {
		log.Printf("[TransactionsService.GetAverageByCategory] ERROR: Failed to fetch earliest date: %v", err)
		return nil, fmt.Errorf("failed to fetch earliest transaction date: %w", err)
	}

	var rangeStart time.Time
	if startDate != nil {
		rangeStart = time.Date(startDate.Year(), startDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else if earliestDate != nil {
		rangeStart = time.Date(earliestDate.Year(), earliestDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		rangeStart = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	var rangeEnd time.Time
	if endDate != nil {
		rangeEnd = time.Date(endDate.Year(), endDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		now := time.Now()
		rangeEnd = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}

	totalMonths := inclusiveMonthCount(rangeStart, rangeEnd)
	if totalMonths < 1 {
		totalMonths = 1
	}

	summaries, err := s.transactionRepo.FindExpenseSummaryByCategory(startDate, endDate)
	if err != nil {
		log.Printf("[TransactionsService.GetAverageByCategory] ERROR: Failed to fetch summaries: %v", err)
		return nil, fmt.Errorf("failed to fetch expense summaries by category: %w", err)
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

	recurringExpenses, err := s.transactionRepo.FindRecurringExpensesInRange(startDate, endDate)
	if err != nil {
		log.Printf("[TransactionsService.GetAverageByCategory] ERROR: Failed to fetch recurring expenses: %v", err)
		return nil, fmt.Errorf("failed to fetch recurring expense summaries by category: %w", err)
	}

	for _, tx := range recurringExpenses {
		if tx.CategoryID == nil || tx.StartDate == nil {
			continue
		}

		txStart := time.Date(tx.StartDate.Year(), tx.StartDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		activeStart := rangeStart
		if txStart.After(activeStart) {
			activeStart = txStart
		}

		activeEnd := rangeEnd
		if tx.EndDate != nil {
			txEnd := time.Date(tx.EndDate.Year(), tx.EndDate.Month(), 1, 0, 0, 0, 0, time.UTC)
			if txEnd.Before(activeEnd) {
				activeEnd = txEnd
			}
		}

		months := inclusiveMonthCount(activeStart, activeEnd)
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

	var result []AverageByCategory
	for catID, info := range categoryMap {
		result = append(result, AverageByCategory{
			CategoryID:   catID,
			CategoryName: info.name,
			Average:      info.total / float64(totalMonths),
			TotalSpent:   info.total,
		})
	}

	return result, nil
}

func (s *transactionsService) GetTransactionsByDateRange(ctx context.Context, startDate, endDate time.Time) ([]model.Transaction, error) {
	return s.transactionRepo.FindByDateRange(startDate, endDate)
}

func (s *transactionsService) GetTransactionsByType(ctx context.Context, transactionType string, limit, offset int) ([]model.Transaction, error) {
	return s.transactionRepo.FindByType(transactionType, limit, offset)
}

func (s *transactionsService) GetRecurringTransactions(ctx context.Context) ([]model.Transaction, error) {
	return s.transactionRepo.FindRecurring()
}

func (s *transactionsService) GetTransactionsByCategory(ctx context.Context, categoryID uint, limit, offset int) ([]model.Transaction, error) {
	return s.transactionRepo.FindByCategory(categoryID, limit, offset)
}

func (s *transactionsService) GetTransactionsByCategories(ctx context.Context, categoriesIDs []uint, limit, offset int) ([]model.Transaction, error) {
	return s.transactionRepo.FindByCategories(categoriesIDs, limit, offset)
}

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

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(original.EndDate.Year(), original.EndDate.Month(), 1, 0, 0, 0, 0, time.UTC)

	if !endDate.After(today) {
		return nil, errors.New("recurring transaction has already ended")
	}

	remainingMonths := exclusiveMonthCount(today, endDate)
	if remainingMonths <= 0 {
		return nil, errors.New("no remaining installments to prepay")
	}

	prepaidAmount := original.Amount * float64(remainingMonths)

	originalDesc := ""
	if original.Description != nil {
		originalDesc = *original.Description
	}
	prepayDesc := fmt.Sprintf("Adiantamento de %d parcelas de %s", remainingMonths, originalDesc)

	newEndDate := time.Now()
	original.EndDate = &newEndDate

	prepayment := &model.Transaction{
		IsRecurring:   false,
		CategoryID:    original.CategoryID,
		CreatedById:   original.CreatedById,
		Amount:        prepaidAmount,
		Type:          original.Type,
		Subtype:       original.Subtype,
		Description:   &prepayDesc,
		Date:          &now,
		PrepaidFromID: &original.ID,
	}

	if err := s.transactionRepo.PrepayTransaction(original, prepayment); err != nil {
		return nil, err
	}

	return &PrepayResult{
		OriginalTransaction: original,
		PrepayTransaction:   prepayment,
		RemainingMonths:     remainingMonths,
		PrepaidAmount:       prepaidAmount,
	}, nil
}

// inclusiveMonthCount returns the number of calendar months spanned, counting both endpoints.
// e.g. Jan → Jan = 1, Jan → Mar = 3.
func inclusiveMonthCount(start, end time.Time) int {
	years := end.Year() - start.Year()
	months := int(end.Month()) - int(start.Month())
	return years*12 + months + 1
}

// exclusiveMonthCount returns the number of months strictly after start and up to end.
// Used when the current month is already paid and only future months remain.
func exclusiveMonthCount(start, end time.Time) int {
	return inclusiveMonthCount(start, end) - 1
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

	if tx.CategoryID != nil {
		categoryTotal, err := s.transactionRepo.FindCurrentMonthTotalByTypeAndCategory(tx.Type, *tx.CategoryID)
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
