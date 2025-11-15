package service

import (
	"context"
	"time"

	"github.com/ArthurWerle/transactions/internal/model"
	"github.com/ArthurWerle/transactions/internal/repository"
)

type TransactionsService interface {
	CreateTransaction(ctx context.Context, transaction *model.Transaction) error
	GetTransactionByID(ctx context.Context, id uint) (*model.Transaction, error)
	GetTransactions(ctx context.Context) ([]model.Transaction, error)
	UpdateTransaction(ctx context.Context, transaction *model.Transaction) error
	DeleteTransaction(ctx context.Context, id uint) error
	GetTransactionsByDateRange(ctx context.Context, startDate, endDate time.Time) ([]model.Transaction, error)
	GetTransactionsByType(ctx context.Context, transactionType string, limit, offset int) ([]model.Transaction, error)
	GetRecurringTransactions(ctx context.Context) ([]model.Transaction, error)
	GetTransactionsByCategory(ctx context.Context, categoryID uint, limit, offset int) ([]model.Transaction, error)
	GetTransactionsByCategories(ctx context.Context, categoriesIDs []uint, limit, offset int) ([]model.Transaction, error)
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

func (s *transactionsService) UpdateTransaction(ctx context.Context, transaction *model.Transaction) error {
	return s.transactionRepo.Update(transaction)
}

func (s *transactionsService) DeleteTransaction(ctx context.Context, id uint) error {
	return s.transactionRepo.Delete(id)
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
