package repository

import (
	"time"

	"github.com/ArthurWerle/transactions/internal/model"
	"gorm.io/gorm"
)

type TransactionsRepository interface {
	Create(transaction *model.Transactions) error
	FindByID(id uint) (*model.Transactions, error)
	FindAll(limit, offset int) ([]model.Transactions, error)
	Update(transaction *model.Transactions) error
	Delete(id uint) error
	FindByDateRange(startDate, endDate time.Time) ([]model.Transactions, error)
	FindByType(transactionType string, limit, offset int) ([]model.Transactions, error)
	FindRecurring() ([]model.Transactions, error)
	FindByCategory(categoryID uint, limit, offset int) ([]model.Transactions, error)
}

type transactionsRepository struct {
	db *gorm.DB
}

func NewTransactionsRepository(db *gorm.DB) TransactionsRepository {
	return &transactionsRepository{db: db}
}

func (r *transactionsRepository) Create(transaction *model.Transactions) error {
	return r.db.Create(transaction).Error
}

func (r *transactionsRepository) FindByID(id uint) (*model.Transactions, error) {
	var transaction model.Transactions
	err := r.db.First(&transaction, id).Error
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func (r *transactionsRepository) FindAll(limit, offset int) ([]model.Transactions, error) {
	var transactions []model.Transactions
	err := r.db.Limit(limit).Offset(offset).Order("date DESC").Find(&transactions).Error
	return transactions, err
}

func (r *transactionsRepository) Update(transaction *model.Transactions) error {
	return r.db.Save(transaction).Error
}

func (r *transactionsRepository) Delete(id uint) error {
	return r.db.Delete(&model.Transactions{}, id).Error
}

func (r *transactionsRepository) FindByDateRange(startDate, endDate time.Time) ([]model.Transactions, error) {
	var transactions []model.Transactions
	err := r.db.Where("date >= ? AND date <= ?", startDate, endDate).
		Order("date DESC").
		Find(&transactions).Error
	return transactions, err
}

func (r *transactionsRepository) FindByType(transactionType string, limit, offset int) ([]model.Transactions, error) {
	var transactions []model.Transactions
	err := r.db.Where("type = ?", transactionType).
		Limit(limit).
		Offset(offset).
		Order("date DESC").
		Find(&transactions).Error
	return transactions, err
}

func (r *transactionsRepository) FindRecurring() ([]model.Transactions, error) {
	var transactions []model.Transactions
	err := r.db.Where("is_recurring = ?", true).
		Order("start_date DESC").
		Find(&transactions).Error
	return transactions, err
}

func (r *transactionsRepository) FindByCategory(categoryID uint, limit, offset int) ([]model.Transactions, error) {
	var transactions []model.Transactions
	err := r.db.Where("category_id = ?", categoryID).
		Limit(limit).
		Offset(offset).
		Order("date DESC").
		Find(&transactions).Error
	return transactions, err
}
