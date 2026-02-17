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

type TransactionsService interface {
	CreateTransaction(ctx context.Context, transaction *model.Transaction) error
	GetAverageByType(ctx context.Context) ([]AverageType, error)
	GetTransactionByID(ctx context.Context, id uint) (*model.Transaction, error)
	GetTransactions(ctx context.Context) ([]model.Transaction, error)
	GetTransactionsWithFilters(ctx context.Context, currentMonth bool, categoryIDs []uint) ([]model.Transaction, error)
	GetLatestTransactions(ctx context.Context) ([]model.Transaction, error)
	GetBiggestTransactions(ctx context.Context) ([]model.Transaction, error)
	UpdateTransaction(ctx context.Context, transaction *model.Transaction) error
	DeleteTransaction(ctx context.Context, id uint) error
	GetTransactionsByDateRange(ctx context.Context, startDate, endDate time.Time) ([]model.Transaction, error)
	GetTransactionsByType(ctx context.Context, transactionType string, limit, offset int) ([]model.Transaction, error)
	GetRecurringTransactions(ctx context.Context) ([]model.Transaction, error)
	GetTransactionsByCategory(ctx context.Context, categoryID uint, limit, offset int) ([]model.Transaction, error)
	GetTransactionsByCategories(ctx context.Context, categoriesIDs []uint, limit, offset int) ([]model.Transaction, error)
	PrepayTransaction(ctx context.Context, id uint) (*PrepayResult, error)
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

func (s *transactionsService) GetTransactions(ctx context.Context) ([]model.Transaction, error) {
	return s.transactionRepo.FindAll()
}

func (s *transactionsService) GetLatestTransactions(ctx context.Context) ([]model.Transaction, error) {
	return s.transactionRepo.FindLatest()
}

func (s *transactionsService) GetBiggestTransactions(ctx context.Context) ([]model.Transaction, error) {
	return s.transactionRepo.FindBiggest()
}

func (s *transactionsService) GetTransactionsWithFilters(ctx context.Context, currentMonth bool, categoryIDs []uint) ([]model.Transaction, error) {
	return s.transactionRepo.FindAllWithFilters(currentMonth, categoryIDs)
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

func (s *transactionsService) GetAverageByType(ctx context.Context) ([]AverageType, error) {
	transactions, err := s.transactionRepo.FindAll()
	if err != nil {
		log.Printf("[TypeService.GetAverageByType] ERROR: Failed to fetch transactions: %v", err)
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	// Group monthly sums by type
	// Key: "type-year-month", Value: sum for that month
	monthlySumsByTypeMonth := make(map[string]float64)

	for _, tx := range transactions {
		if tx.Date == nil {
			continue
		}
		monthKey := fmt.Sprintf("%s-%d-%d",
			tx.Type,
			tx.Date.Year(),
			tx.Date.Month())

		monthlySumsByTypeMonth[monthKey] += tx.Amount
	}

	// Group monthly sums by type only (to calculate average across months)
	monthlySumsByType := make(map[string][]float64)

	for key, sum := range monthlySumsByTypeMonth {
		// Extract type name (everything before the first dash followed by a digit)
		typeName := extractTypeName(key)
		monthlySumsByType[typeName] = append(monthlySumsByType[typeName], sum)
	}

	var result []AverageType
	for typeName, monthlySums := range monthlySumsByType {
		var total float64
		for _, sum := range monthlySums {
			total += sum
		}
		average := total / float64(len(monthlySums))

		result = append(result, AverageType{
			TypeName: typeName,
			Average:  average,
		})
	}

	return result, nil
}

func extractTypeName(key string) string {
	// Find the position where the year starts (first digit after a dash)
	for i := 0; i < len(key); i++ {
		if key[i] == '-' && i+1 < len(key) && key[i+1] >= '0' && key[i+1] <= '9' {
			return key[:i]
		}
	}
	return key
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
	alreadyPrepaid, err := s.transactionRepo.FindByPrepaidID(id)
	if err != nil {
		return nil, err
	}
	if alreadyPrepaid != nil {
		return nil, errors.New("transaction was already prepaid")
	}

	original, err := s.transactionRepo.FindByID(id)
	if err != nil {
		return nil, err
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

	remainingMonths := monthsBetween(today, endDate)
	if remainingMonths <= 0 {
		return nil, errors.New("no remaining installments to prepay")
	}

	prepaidAmount := original.Amount * float64(remainingMonths)

	originalDesc := ""
	if original.Description != nil {
		originalDesc = *original.Description
	}
	prepayDesc := fmt.Sprintf("Adiantamento de %d parcelas de %s", remainingMonths, originalDesc)

	newEndDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
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

func monthsBetween(start, end time.Time) int {
	years := end.Year() - start.Year()
	months := int(end.Month()) - int(start.Month())
	total := years*12 + months
	return total
}
