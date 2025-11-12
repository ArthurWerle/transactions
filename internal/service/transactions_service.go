package service

import (
	"context"

	"github.com/ArthurWerle/transactions/internal/model"
	"github.com/ArthurWerle/transactions/internal/repository"
	"gorm.io/gorm"
)

type TransactionsService interface {
	GetTransactions(ctx context.Context) ([]model.Transaction, error)
}

type transactionsService struct {
	transactionRepo repository.TransactionsRepository
}

func NewTransactionsService(transactionRepo repository.TransactionsRepository) TransactionsService {
	return &transactionsService{
		transactionRepo: transactionRepo,
	}
}

func (s *transactionsService) GetTransactions(ctx context.Context, transaction *model.Transaction) ([]model.Transaction, error) {
	transactions, err := s.transactionRepo.FindAll()
	if err != nil {
		return nil, err
	}

	return transactions, nil
}
