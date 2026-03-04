package repository

import (
	"time"

	"github.com/ArthurWerle/transactions/internal/model"
	"gorm.io/gorm"
)

type CategoryExpenseSummary struct {
	CategoryID   *uint   `gorm:"column:category_id"`
	CategoryName string  `gorm:"column:category_name"`
	TotalSpent   float64 `gorm:"column:total_spent"`
}

type RecurringCategoryExpense struct {
	CategoryID   *uint      `gorm:"column:category_id"`
	CategoryName string     `gorm:"column:category_name"`
	Amount       float64    `gorm:"column:amount"`
	StartDate    *time.Time `gorm:"column:start_date"`
	EndDate      *time.Time `gorm:"column:end_date"`
}

type TransactionsRepository interface {
	Create(transaction *model.Transaction) error
	FindByID(id uint) (*model.Transaction, error)
	FindByPrepaidID(prepaidID uint) (*model.Transaction, error)
	FindAll() ([]model.Transaction, error)
	FindAllWithFilters(currentMonth bool, categoryIDs []uint, searchQuery string, startDate, endDate *time.Time, transactionType string) ([]model.Transaction, error)
	FindBiggest(month, year int) ([]model.Transaction, error)
	FindLatest() ([]model.Transaction, error)
	Update(transaction *model.Transaction) error
	Delete(id uint) error
	FindByDateRange(startDate, endDate time.Time) ([]model.Transaction, error)
	FindByType(transactionType string, limit, offset int) ([]model.Transaction, error)
	FindRecurring() ([]model.Transaction, error)
	FindByCategories(categoriesIDs []uint, limit, offset int) ([]model.Transaction, error)
	FindByCategory(categoryID uint, limit, offset int) ([]model.Transaction, error)
	PrepayTransaction(original *model.Transaction, prepayment *model.Transaction) error
	FindEarliestDate(startDate, endDate *time.Time) (*time.Time, error)
	FindExpenseSummaryByCategory(startDate, endDate *time.Time) ([]CategoryExpenseSummary, error)
	FindRecurringExpensesInRange(startDate, endDate *time.Time) ([]RecurringCategoryExpense, error)
	FindCurrentMonthTotalByType(transactionType string) (float64, error)
	FindCurrentMonthTotalByTypeAndCategory(transactionType string, categoryID uint) (float64, error)
}

type transactionsRepository struct {
	db *gorm.DB
}

func NewTransactionsRepository(db *gorm.DB) TransactionsRepository {
	return &transactionsRepository{db: db}
}

func WithIsPrepaid(db *gorm.DB) *gorm.DB {
	return db.Select(`*, EXISTS(
		SELECT 1 FROM transactions t2
		WHERE t2.prepaid_from_id = transactions.id
		AND t2.deleted_at IS NULL
	) AS is_prepaid`)
}

func (r *transactionsRepository) Create(transaction *model.Transaction) error {
	return r.db.Create(transaction).Error
}

func (r *transactionsRepository) FindByID(id uint) (*model.Transaction, error) {
	var transaction model.Transaction
	err := r.db.Scopes(WithIsPrepaid).First(&transaction, id).Error
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func (r *transactionsRepository) FindByPrepaidID(prepaidID uint) (*model.Transaction, error) {
	var transaction model.Transaction
	err := r.db.Scopes(WithIsPrepaid).Where("prepaid_from_id = ?", prepaidID).First(&transaction).Error
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func (r *transactionsRepository) FindAll() ([]model.Transaction, error) {
	var transactions []model.Transaction
	err := r.db.Scopes(WithIsPrepaid).Order("is_recurring, date DESC").Find(&transactions).Error
	return transactions, err
}

func (r *transactionsRepository) FindAllWithFilters(currentMonth bool, categoryIDs []uint, searchQuery string, startDate, endDate *time.Time, transactionType string) ([]model.Transaction, error) {
	var transactions []model.Transaction
	query := r.db.Model(&model.Transaction{}).Scopes(WithIsPrepaid)

	if currentMonth {
		now := time.Now()
		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Second)
		query = query.Where("(date >= ? AND date <= ?) OR (is_recurring = true AND start_date <= ? AND (end_date >= ? OR end_date IS NULL))", startOfMonth, endOfMonth, endOfMonth, startOfMonth)
	} else if startDate != nil || endDate != nil {
		sd := time.Time{}
		if startDate != nil {
			sd = *startDate
		}
		ed := time.Now()
		if endDate != nil {
			ed = *endDate
		}
		query = query.Where(
			"(is_recurring = ? AND date >= ? AND date <= ?) OR (is_recurring = ? AND start_date <= ? AND (end_date >= ? OR end_date IS NULL))",
			false, sd, ed,
			true, ed, sd,
		)
	}

	if len(categoryIDs) > 0 {
		query = query.Where("category_id IN ?", categoryIDs)
	}

	if searchQuery != "" {
		query = query.Where("description ILIKE ?", "%"+searchQuery+"%")
	}

	if transactionType != "" {
		query = query.Where("type = ?", transactionType)
	}

	err := query.Order("is_recurring, date DESC").Find(&transactions).Error
	return transactions, err
}

func (r *transactionsRepository) FindLatest() ([]model.Transaction, error) {
	var transactions []model.Transaction
	err := r.db.Model(&model.Transaction{}).Scopes(WithIsPrepaid).Where("date IS NOT NULL AND type = ?", model.Expense).Order("date DESC").Limit(3).Find(&transactions).Error
	return transactions, err
}

func (r *transactionsRepository) FindBiggest(month, year int) ([]model.Transaction, error) {
	var transactions []model.Transaction
	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Second)
	err := r.db.Model(&model.Transaction{}).Scopes(WithIsPrepaid).Where(
		"type = ? AND ((is_recurring = false AND date >= ? AND date <= ?) OR (is_recurring = true AND start_date <= ? AND (end_date >= ? OR end_date IS NULL)))",
		model.Expense, startOfMonth, endOfMonth, endOfMonth, startOfMonth,
	).Order("amount DESC").Limit(3).Find(&transactions).Error
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
	err := r.db.Scopes(WithIsPrepaid).Where(
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
	err := r.db.Scopes(WithIsPrepaid).Where("type = ?", transactionType).
		Limit(limit).
		Offset(offset).
		Order("date DESC").
		Find(&transactions).Error
	return transactions, err
}

func (r *transactionsRepository) FindRecurring() ([]model.Transaction, error) {
	var transactions []model.Transaction
	err := r.db.Scopes(WithIsPrepaid).Where("is_recurring = ?", true).
		Order("start_date DESC").
		Find(&transactions).Error
	return transactions, err
}

func (r *transactionsRepository) FindByCategory(categoryID uint, limit, offset int) ([]model.Transaction, error) {
	var transactions []model.Transaction
	err := r.db.Scopes(WithIsPrepaid).Where("category_id = ?", categoryID).
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

	err := r.db.Scopes(WithIsPrepaid).Where("category_id in ?", categoriesIDs).
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

func (r *transactionsRepository) FindEarliestDate(startDate, endDate *time.Time) (*time.Time, error) {
	var result struct {
		MinDate *time.Time `gorm:"column:min_date"`
	}
	query := "SELECT MIN(date) as min_date FROM transactions WHERE date IS NOT NULL AND deleted_at IS NULL"
	args := []interface{}{}
	if startDate != nil {
		query += " AND date >= ?"
		args = append(args, *startDate)
	}
	if endDate != nil {
		query += " AND date <= ?"
		args = append(args, *endDate)
	}
	err := r.db.Raw(query, args...).Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return result.MinDate, nil
}

func (r *transactionsRepository) FindExpenseSummaryByCategory(startDate, endDate *time.Time) ([]CategoryExpenseSummary, error) {
	var results []CategoryExpenseSummary
	query := `
		SELECT
			t.category_id,
			c.name as category_name,
			SUM(t.amount) as total_spent
		FROM transactions t
		LEFT JOIN categories c ON c.id = t.category_id
		WHERE t.type = 'expense'
			AND t.category_id IS NOT NULL
			AND t.date IS NOT NULL
			AND t.deleted_at IS NULL`
	args := []interface{}{}
	if startDate != nil {
		query += " AND t.date >= ?"
		args = append(args, *startDate)
	}
	if endDate != nil {
		query += " AND t.date <= ?"
		args = append(args, *endDate)
	}
	query += " GROUP BY t.category_id, c.name"
	err := r.db.Raw(query, args...).Scan(&results).Error
	return results, err
}

func (r *transactionsRepository) FindRecurringExpensesInRange(startDate, endDate *time.Time) ([]RecurringCategoryExpense, error) {
	var results []RecurringCategoryExpense
	query := `
		SELECT
			t.category_id,
			c.name as category_name,
			t.amount,
			t.start_date,
			t.end_date
		FROM transactions t
		LEFT JOIN categories c ON c.id = t.category_id
		WHERE t.type = 'expense'
			AND t.category_id IS NOT NULL
			AND t.is_recurring = true
			AND t.start_date IS NOT NULL
			AND t.deleted_at IS NULL`
	args := []interface{}{}
	if endDate != nil {
		query += " AND t.start_date <= ?"
		args = append(args, *endDate)
	}
	if startDate != nil {
		query += " AND (t.end_date IS NULL OR t.end_date >= ?)"
		args = append(args, *startDate)
	}
	err := r.db.Raw(query, args...).Scan(&results).Error
	return results, err
}

func (r *transactionsRepository) FindCurrentMonthTotalByType(transactionType string) (float64, error) {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Second)

	var result struct {
		Total float64 `gorm:"column:total"`
	}
	err := r.db.Raw(`
		SELECT COALESCE(SUM(amount), 0) as total
		FROM transactions
		WHERE type = ?
		  AND deleted_at IS NULL
		  AND (
		    (is_recurring = false AND date >= ? AND date <= ?)
		    OR
		    (is_recurring = true AND start_date <= ? AND (end_date >= ? OR end_date IS NULL))
		  )
	`, transactionType, startOfMonth, endOfMonth, endOfMonth, startOfMonth).Scan(&result).Error

	return result.Total, err
}

func (r *transactionsRepository) FindCurrentMonthTotalByTypeAndCategory(transactionType string, categoryID uint) (float64, error) {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Second)

	var result struct {
		Total float64 `gorm:"column:total"`
	}
	err := r.db.Raw(`
		SELECT COALESCE(SUM(amount), 0) as total
		FROM transactions
		WHERE type = ?
		  AND category_id = ?
		  AND deleted_at IS NULL
		  AND (
		    (is_recurring = false AND date >= ? AND date <= ?)
		    OR
		    (is_recurring = true AND start_date <= ? AND (end_date >= ? OR end_date IS NULL))
		  )
	`, transactionType, categoryID, startOfMonth, endOfMonth, endOfMonth, startOfMonth).Scan(&result).Error

	return result.Total, err
}
