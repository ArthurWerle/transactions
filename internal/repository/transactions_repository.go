package repository

import (
	"time"

	"github.com/ArthurWerle/transactions/internal/model"
	"gorm.io/gorm"
)

type TransactionsRepository interface {
	Create(transaction *model.Transaction) error
	FindByID(id uint) (*model.Transaction, error)
	FindAll() ([]model.Transaction, error)
	Update(transaction *model.Transaction) error
	Delete(id uint) error
	FindByDateRange(startDate, endDate time.Time) ([]model.Transaction, error)
	FindByType(transactionType string, limit, offset int) ([]model.Transaction, error)
	FindRecurring() ([]model.Transaction, error)
	FindByCategories(categoriesIDs []uint, limit, offset int) ([]model.Transaction, error)
	FindByCategory(categoryID uint, limit, offset int) ([]model.Transaction, error)
}

type transactionsRepository struct {
	db *gorm.DB
}

func NewTransactionsRepository(db *gorm.DB) TransactionsRepository {
	return &transactionsRepository{db: db}
}

func (r *transactionsRepository) Create(transaction *model.Transaction) error {
	return r.db.Create(transaction).Error
}

func (r *transactionsRepository) FindByID(id uint) (*model.Transaction, error) {
	var transaction model.Transaction
	err := r.db.First(&transaction, id).Error
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func (r *transactionsRepository) FindAll() ([]model.Transaction, error) {
	var transactions []model.Transaction
	err := r.db.Order("date DESC").Find(&transactions).Error
	return transactions, err
}

func (r *transactionsRepository) Update(transaction *model.Transaction) error {
	return r.db.Save(transaction).Error
}

func (r *transactionsRepository) Delete(id uint) error {
	return r.db.Delete(&model.Transaction{}, id).Error
}

func (r *transactionsRepository) FindByDateRange(startDate, endDate time.Time) ([]model.Transaction, error) {
	var transactions []model.Transaction
	err := r.db.Where("date >= ? AND date <= ?", startDate, endDate).
		Order("date DESC").
		Find(&transactions).Error
	return transactions, err
}

func (r *transactionsRepository) FindByType(transactionType string, limit, offset int) ([]model.Transaction, error) {
	var transactions []model.Transaction
	err := r.db.Where("type = ?", transactionType).
		Limit(limit).
		Offset(offset).
		Order("date DESC").
		Find(&transactions).Error
	return transactions, err
}

func (r *transactionsRepository) FindRecurring() ([]model.Transaction, error) {
	var transactions []model.Transaction
	err := r.db.Where("is_recurring = ?", true).
		Order("start_date DESC").
		Find(&transactions).Error
	return transactions, err
}

func (r *transactionsRepository) FindByCategory(categoryID uint, limit, offset int) ([]model.Transaction, error) {
	var transactions []model.Transaction
	err := r.db.Where("category_id = ?", categoryID).
		Limit(limit).
		Offset(offset).
		Order("date DESC").
		Find(&transactions).Error
	return transactions, err
}

func (r *transactionsRepository) FindByCategories(categoriesIDs []uint, limit, offset int) ([]model.Transaction, error) {
	var transactions []model.Transaction

	if len(categoriesIDs) == 0 {
		return transactions, nil
	}

	err := r.db.Where("category_id in ?", categoriesIDs).
		Limit(limit).
		Offset(offset).
		Order("date DESC").
		Find(&transactions).Error
	return transactions, err
}
