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
	FindAllWithFilters(currentMonth bool, categoryIDs []uint) ([]model.Transaction, error)
	FindBiggest() ([]model.Transaction, error)
	FindLatest() ([]model.Transaction, error)
	Update(transaction *model.Transaction) error
	Delete(id uint) error
	FindByDateRange(startDate, endDate time.Time) ([]model.Transaction, error)
	FindByType(transactionType string, limit, offset int) ([]model.Transaction, error)
	FindRecurring() ([]model.Transaction, error)
	FindByCategories(categoriesIDs []uint, limit, offset int) ([]model.Transaction, error)
	FindByCategory(categoryID uint, limit, offset int) ([]model.Transaction, error)
	PrepayTransaction(original *model.Transaction, prepayment *model.Transaction) error
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

func (r *transactionsRepository) FindAllWithFilters(currentMonth bool, categoryIDs []uint) ([]model.Transaction, error) {
	var transactions []model.Transaction
	query := r.db.Model(&model.Transaction{})

	if currentMonth {
		now := time.Now()
		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Second)
		query = query.Where("date >= ? AND date <= ?", startOfMonth, endOfMonth)
	}

	if len(categoryIDs) > 0 {
		query = query.Where("category_id IN ?", categoryIDs)
	}

	err := query.Order("date DESC").Find(&transactions).Error
	return transactions, err
}

func (r *transactionsRepository) FindLatest() ([]model.Transaction, error) {
	var transactions []model.Transaction
	query := r.db.Model(&model.Transaction{})
	err := query.Where("date IS NOT NULL AND type = ?", model.Expense).Order("date DESC").Limit(3).Find(&transactions).Error
	return transactions, err
}

func (r *transactionsRepository) FindBiggest() ([]model.Transaction, error) {
	var transactions []model.Transaction
	query := r.db.Model(&model.Transaction{})
	err := query.Where("type = ? AND date_trunc('month', CURRENT_DATE) = date_trunc('month', date)", model.Expense).Order("amount DESC").Limit(3).Find(&transactions).Error
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
	err := r.db.Where(
		"(is_recurring = ? AND date >= ? AND date <= ?) OR (is_recurring = ? AND start_date <= ? AND (end_date >= ? OR end_date IS NULL))",
		false, startDate, endDate,
		true, endDate, startDate,
	).
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

func (r *transactionsRepository) PrepayTransaction(original *model.Transaction, prepayment *model.Transaction) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(original).Error; err != nil {
			return err
		}
		if err := tx.Create(prepayment).Error; err != nil {
			return err
		}
		return nil
	})
}
