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

type RecurringAmount struct {
	Amount    float64    `gorm:"column:amount"`
	StartDate *time.Time `gorm:"column:start_date"`
	EndDate   *time.Time `gorm:"column:end_date"`
}

type MonthlyTypeTotal struct {
	Type       string  `gorm:"column:type"`
	Year       int     `gorm:"column:year"`
	Month      int     `gorm:"column:month"`
	MonthlySum float64 `gorm:"column:monthly_sum"`
}

type RecurringTypeTransaction struct {
	Type      string     `gorm:"column:type"`
	Amount    float64    `gorm:"column:amount"`
	StartDate *time.Time `gorm:"column:start_date"`
	EndDate   *time.Time `gorm:"column:end_date"`
}

type MonthlyFlowRow struct {
	Month   time.Time `gorm:"column:month"`
	Income  float64   `gorm:"column:income"`
	Expense float64   `gorm:"column:expense"`
}

type CategoryMonthlyFlowRow struct {
	Month        time.Time `gorm:"column:month"`
	CategoryID   uint      `gorm:"column:category_id"`
	CategoryName string    `gorm:"column:category_name"`
	Color        string    `gorm:"column:color"`
	Income       float64   `gorm:"column:income"`
	Expense      float64   `gorm:"column:expense"`
}

type CategoryMonthTotal struct {
	CategoryName string  `gorm:"column:category_name"`
	Total        float64 `gorm:"column:total"`
}

type TransactionsRepository interface {
	Create(transaction *model.Transaction) error
	FindByID(id uint) (*model.Transaction, error)
	FindByPrepaidID(prepaidID uint) (*model.Transaction, error)
	FindAll(limit, offset int) ([]model.Transaction, error)
	CountAll() (int64, error)
	FindAllWithFilters(currentMonth bool, categoryIDs []uint, searchQuery string, startDate, endDate *time.Time, transactionType string, limit, offset int) ([]model.Transaction, error)
	CountAllWithFilters(currentMonth bool, categoryIDs []uint, searchQuery string, startDate, endDate *time.Time, transactionType string) (int64, error)
	FindBiggest(month, year int) ([]model.Transaction, error)
	FindLatest() ([]model.Transaction, error)
	Update(transaction *model.Transaction) error
	Delete(id uint) error
	FindByDateRange(startDate, endDate time.Time) ([]model.Transaction, error)
	FindMonthlyFlow(startMonth, endMonth time.Time) ([]MonthlyFlowRow, error)
	FindCategoryMonthlyFlow(startMonth, endMonth time.Time, categoryIDs []uint) ([]CategoryMonthlyFlowRow, error)
	FindCategoryExpenseTotalsForMonth(month time.Time) ([]CategoryMonthTotal, error)
	FindEarliestDate(startDate, endDate *time.Time) (*time.Time, error)
	FindExpenseSummaryByCategory(startDate, endDate *time.Time) ([]CategoryExpenseSummary, error)
	FindRecurringExpensesInRange(startDate, endDate *time.Time) ([]RecurringCategoryExpense, error)
	FindIncomeTotalInRange(startDate, endDate *time.Time) (float64, error)
	FindRecurringIncomesInRange(startDate, endDate *time.Time) ([]RecurringAmount, error)
	FindCurrentMonthTotalByType(transactionType string) (float64, error)
	FindCurrentMonthTotalByTypeAndCategory(transactionType string, categoryID uint) (float64, error)
	FindNonRecurringMonthlyTotalsByType() ([]MonthlyTypeTotal, error)
	FindRecurringTransactionSummaryByType() ([]RecurringTypeTransaction, error)
}

type transactionsRepository struct {
	db  *gorm.DB
	loc *time.Location
}

func NewTransactionsRepository(db *gorm.DB, loc *time.Location) TransactionsRepository {
	return &transactionsRepository{db: db, loc: loc}
}

func WithIsPrepaid(db *gorm.DB) *gorm.DB {
	return db.Select(`*, EXISTS(
		SELECT 1 FROM transactions t2
		WHERE t2.prepaid_from_id = transactions.id
		AND t2.deleted_at IS NULL
	) AS is_prepaid`)
}

// monthBounds returns the half-open interval [first instant of the month,
// first instant of the next month) in the given location.
func monthBounds(year int, month time.Month, loc *time.Location) (time.Time, time.Time) {
	start := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	return start, start.AddDate(0, 1, 0)
}

// periodCondition returns a WHERE fragment matching transactions active in the
// half-open interval [start, end): one-off transactions by their timestamp,
// recurring ones by schedule overlap. A nil bound is unbounded, so a
// start-only filter also matches recurring schedules that begin in the
// future. start_date/end_date are DATE columns, so the recurring legs compare
// against the bound's calendar date in the reporting timezone.
func (r *transactionsRepository) periodCondition(start, end *time.Time) (string, []interface{}) {
	nonRecurring := "is_recurring = false AND date IS NOT NULL"
	args := []interface{}{}
	if start != nil {
		nonRecurring += " AND date >= ?"
		args = append(args, *start)
	}
	if end != nil {
		nonRecurring += " AND date < ?"
		args = append(args, *end)
	}

	recurring := "is_recurring = true AND start_date IS NOT NULL"
	if end != nil {
		recurring += " AND start_date < ?"
		args = append(args, end.In(r.loc).Format("2006-01-02"))
	}
	if start != nil {
		recurring += " AND (end_date IS NULL OR end_date >= ?)"
		args = append(args, start.In(r.loc).Format("2006-01-02"))
	}

	return "((" + nonRecurring + ") OR (" + recurring + "))", args
}

func (r *transactionsRepository) Create(transaction *model.Transaction) error {
	return r.db.Create(transaction).Error
}

func (r *transactionsRepository) FindByID(id uint) (*model.Transaction, error) {
	var transaction model.Transaction
	err := r.db.Scopes(WithIsPrepaid).Preload("Subcategory").Preload("Location").First(&transaction, id).Error
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func (r *transactionsRepository) FindByPrepaidID(prepaidID uint) (*model.Transaction, error) {
	var transaction model.Transaction
	err := r.db.Scopes(WithIsPrepaid).Preload("Subcategory").Preload("Location").Where("prepaid_from_id = ?", prepaidID).First(&transaction).Error
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func (r *transactionsRepository) FindAll(limit, offset int) ([]model.Transaction, error) {
	var transactions []model.Transaction
	err := r.db.Scopes(WithIsPrepaid).Preload("Subcategory").Preload("Location").Order("is_recurring, date DESC").Limit(limit).Offset(offset).Find(&transactions).Error
	return transactions, err
}

func (r *transactionsRepository) CountAll() (int64, error) {
	var count int64
	err := r.db.Model(&model.Transaction{}).Count(&count).Error
	return count, err
}

func (r *transactionsRepository) buildFilterQuery(currentMonth bool, categoryIDs []uint, searchQuery string, startDate, endDate *time.Time, transactionType string) *gorm.DB {
	query := r.db.Model(&model.Transaction{})

	if currentMonth {
		now := time.Now().In(r.loc)
		start, end := monthBounds(now.Year(), now.Month(), r.loc)
		cond, args := r.periodCondition(&start, &end)
		query = query.Where(cond, args...)
	} else if startDate != nil || endDate != nil {
		cond, args := r.periodCondition(startDate, endDate)
		query = query.Where(cond, args...)
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

	return query
}

func (r *transactionsRepository) FindAllWithFilters(currentMonth bool, categoryIDs []uint, searchQuery string, startDate, endDate *time.Time, transactionType string, limit, offset int) ([]model.Transaction, error) {
	var transactions []model.Transaction
	query := r.buildFilterQuery(currentMonth, categoryIDs, searchQuery, startDate, endDate, transactionType).Scopes(WithIsPrepaid)
	err := query.Preload("Subcategory").Preload("Location").Order("is_recurring, date DESC").Limit(limit).Offset(offset).Find(&transactions).Error
	return transactions, err
}

func (r *transactionsRepository) CountAllWithFilters(currentMonth bool, categoryIDs []uint, searchQuery string, startDate, endDate *time.Time, transactionType string) (int64, error) {
	var count int64
	err := r.buildFilterQuery(currentMonth, categoryIDs, searchQuery, startDate, endDate, transactionType).Count(&count).Error
	return count, err
}


func (r *transactionsRepository) FindLatest() ([]model.Transaction, error) {
	var transactions []model.Transaction
	err := r.db.Model(&model.Transaction{}).Scopes(WithIsPrepaid).Preload("Subcategory").Preload("Location").Where("date IS NOT NULL AND type = ?", model.Expense).Order("date DESC").Limit(3).Find(&transactions).Error
	return transactions, err
}

func (r *transactionsRepository) FindBiggest(month, year int) ([]model.Transaction, error) {
	var transactions []model.Transaction
	start, end := monthBounds(year, time.Month(month), r.loc)
	cond, args := r.periodCondition(&start, &end)
	err := r.db.Model(&model.Transaction{}).Scopes(WithIsPrepaid).Preload("Subcategory").Preload("Location").
		Where("type = ?", model.Expense).
		Where("prepaid_from_id IS NULL").
		Where(cond, args...).
		Order("amount DESC").Limit(3).Find(&transactions).Error
	return transactions, err
}

func (r *transactionsRepository) Update(transaction *model.Transaction) error {
	return r.db.Save(transaction).Error
}

func (r *transactionsRepository) Delete(id uint) error {
	return r.db.Delete(&model.Transaction{}, id).Error
}

// FindByDateRange treats endDate as exclusive; callers convert inclusive
// calendar days to the half-open form before calling.
func (r *transactionsRepository) FindByDateRange(startDate, endDate time.Time) ([]model.Transaction, error) {
	var transactions []model.Transaction
	cond, args := r.periodCondition(&startDate, &endDate)
	err := r.db.Scopes(WithIsPrepaid).Preload("Subcategory").Preload("Location").
		Where(cond, args...).
		Order("date DESC").
		Find(&transactions).Error
	return transactions, err
}

// FindMonthlyFlow returns income/expense totals for every calendar month in
// [startMonth, endMonth] (inclusive, zero-filled) in one query: one-offs are
// bucketed by their reporting-timezone month, recurring schedules contribute
// their amount to each month they are active in. Prepayment rows are cash
// records excluded from aggregates (consumption basis).
func (r *transactionsRepository) FindMonthlyFlow(startMonth, endMonth time.Time) ([]MonthlyFlowRow, error) {
	var results []MonthlyFlowRow
	tz := r.loc.String()
	err := r.db.Raw(`
		WITH months AS (
			SELECT generate_series(?::date, ?::date, interval '1 month')::date AS month
		),
		activity AS (
			SELECT date_trunc('month', date AT TIME ZONE ?)::date AS month, type, amount
			FROM transactions
			WHERE is_recurring = false
			  AND date IS NOT NULL
			  AND deleted_at IS NULL
			  AND prepaid_from_id IS NULL
			UNION ALL
			SELECT m.month, t.type, t.amount
			FROM months m
			JOIN transactions t ON t.is_recurring = true
			  AND t.deleted_at IS NULL
			  AND t.start_date < (m.month + interval '1 month')::date
			  AND (t.end_date IS NULL OR t.end_date >= m.month)
		)
		SELECT m.month,
		       COALESCE(SUM(a.amount) FILTER (WHERE a.type = 'income'), 0)  AS income,
		       COALESCE(SUM(a.amount) FILTER (WHERE a.type = 'expense'), 0) AS expense
		FROM months m
		LEFT JOIN activity a ON a.month = m.month
		GROUP BY m.month
		ORDER BY m.month
	`, startMonth.Format("2006-01-02"), endMonth.Format("2006-01-02"), tz).Scan(&results).Error
	return results, err
}

// FindCategoryMonthlyFlow returns per-category income/expense totals for each
// month in [startMonth, endMonth] where the category had activity, in one
// query. Soft-deleted categories stay visible, labeled.
func (r *transactionsRepository) FindCategoryMonthlyFlow(startMonth, endMonth time.Time, categoryIDs []uint) ([]CategoryMonthlyFlowRow, error) {
	var results []CategoryMonthlyFlowRow
	tz := r.loc.String()

	query := `
		WITH months AS (
			SELECT generate_series(?::date, ?::date, interval '1 month')::date AS month
		),
		activity AS (
			SELECT date_trunc('month', t.date AT TIME ZONE ?)::date AS month, t.category_id, t.type, t.amount
			FROM transactions t
			WHERE t.is_recurring = false
			  AND t.date IS NOT NULL
			  AND t.deleted_at IS NULL
			  AND t.prepaid_from_id IS NULL
			  AND t.date >= (SELECT min(month) FROM months)
			  AND t.date < ((SELECT max(month) FROM months) + interval '1 month')
			UNION ALL
			SELECT m.month, t.category_id, t.type, t.amount
			FROM months m
			JOIN transactions t ON t.is_recurring = true
			  AND t.deleted_at IS NULL
			  AND t.start_date < (m.month + interval '1 month')::date
			  AND (t.end_date IS NULL OR t.end_date >= m.month)
		)
		SELECT a.month,
		       a.category_id,
		       CASE WHEN c.deleted_at IS NOT NULL THEN c.name || ' (deleted)' ELSE c.name END AS category_name,
		       COALESCE(c.color, '') AS color,
		       COALESCE(SUM(a.amount) FILTER (WHERE a.type = 'income'), 0)  AS income,
		       COALESCE(SUM(a.amount) FILTER (WHERE a.type = 'expense'), 0) AS expense
		FROM activity a
		JOIN categories c ON c.id = a.category_id`
	args := []interface{}{startMonth.Format("2006-01-02"), endMonth.Format("2006-01-02"), tz}
	if len(categoryIDs) > 0 {
		query += " WHERE a.category_id IN ?"
		args = append(args, categoryIDs)
	}
	query += `
		GROUP BY a.month, a.category_id, c.name, c.deleted_at, c.color
		ORDER BY category_name, a.month`

	err := r.db.Raw(query, args...).Scan(&results).Error
	return results, err
}

// FindCategoryExpenseTotalsForMonth returns each category's expense total for
// one calendar month, largest first, with soft-deleted categories labeled.
func (r *transactionsRepository) FindCategoryExpenseTotalsForMonth(month time.Time) ([]CategoryMonthTotal, error) {
	var results []CategoryMonthTotal
	tz := r.loc.String()
	monthStr := month.Format("2006-01-02")
	err := r.db.Raw(`
		WITH activity AS (
			SELECT t.category_id, t.amount
			FROM transactions t
			WHERE t.is_recurring = false
			  AND t.type = 'expense'
			  AND t.date IS NOT NULL
			  AND t.deleted_at IS NULL
			  AND t.prepaid_from_id IS NULL
			  AND date_trunc('month', t.date AT TIME ZONE ?)::date = ?::date
			UNION ALL
			SELECT t.category_id, t.amount
			FROM transactions t
			WHERE t.is_recurring = true
			  AND t.type = 'expense'
			  AND t.deleted_at IS NULL
			  AND t.start_date < (?::date + interval '1 month')::date
			  AND (t.end_date IS NULL OR t.end_date >= ?::date)
		)
		SELECT CASE WHEN c.deleted_at IS NOT NULL THEN c.name || ' (deleted)' ELSE c.name END AS category_name,
		       SUM(a.amount) AS total
		FROM activity a
		JOIN categories c ON c.id = a.category_id
		GROUP BY c.name, c.deleted_at
		ORDER BY total DESC
	`, tz, monthStr, monthStr, monthStr).Scan(&results).Error
	return results, err
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
		query += " AND date < ?"
		args = append(args, endDate.AddDate(0, 0, 1))
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
			CASE WHEN c.deleted_at IS NOT NULL THEN c.name || ' (deleted)' ELSE c.name END as category_name,
			SUM(t.amount) as total_spent
		FROM transactions t
		LEFT JOIN categories c ON c.id = t.category_id
		WHERE t.type = 'expense'
			AND t.category_id IS NOT NULL
			AND t.date IS NOT NULL
			AND t.deleted_at IS NULL
			AND t.prepaid_from_id IS NULL`
	args := []interface{}{}
	if startDate != nil {
		query += " AND t.date >= ?"
		args = append(args, *startDate)
	}
	if endDate != nil {
		query += " AND t.date < ?"
		args = append(args, endDate.AddDate(0, 0, 1))
	}
	query += " GROUP BY t.category_id, c.name, c.deleted_at"
	err := r.db.Raw(query, args...).Scan(&results).Error
	return results, err
}

func (r *transactionsRepository) FindRecurringExpensesInRange(startDate, endDate *time.Time) ([]RecurringCategoryExpense, error) {
	var results []RecurringCategoryExpense
	query := `
		SELECT
			t.category_id,
			CASE WHEN c.deleted_at IS NOT NULL THEN c.name || ' (deleted)' ELSE c.name END as category_name,
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
		args = append(args, endDate.In(r.loc).Format("2006-01-02"))
	}
	if startDate != nil {
		query += " AND (t.end_date IS NULL OR t.end_date >= ?)"
		args = append(args, startDate.In(r.loc).Format("2006-01-02"))
	}
	err := r.db.Raw(query, args...).Scan(&results).Error
	return results, err
}

func (r *transactionsRepository) FindIncomeTotalInRange(startDate, endDate *time.Time) (float64, error) {
	var result struct {
		Total float64 `gorm:"column:total"`
	}
	query := `
		SELECT COALESCE(SUM(amount), 0) as total
		FROM transactions
		WHERE type = 'income'
		  AND is_recurring = false
		  AND date IS NOT NULL
		  AND deleted_at IS NULL
		  AND prepaid_from_id IS NULL`
	args := []interface{}{}
	if startDate != nil {
		query += " AND date >= ?"
		args = append(args, *startDate)
	}
	if endDate != nil {
		query += " AND date < ?"
		args = append(args, endDate.AddDate(0, 0, 1))
	}
	err := r.db.Raw(query, args...).Scan(&result).Error
	return result.Total, err
}

func (r *transactionsRepository) FindRecurringIncomesInRange(startDate, endDate *time.Time) ([]RecurringAmount, error) {
	var results []RecurringAmount
	query := `
		SELECT amount, start_date, end_date
		FROM transactions
		WHERE type = 'income'
		  AND is_recurring = true
		  AND start_date IS NOT NULL
		  AND deleted_at IS NULL`
	args := []interface{}{}
	if endDate != nil {
		query += " AND start_date <= ?"
		args = append(args, endDate.In(r.loc).Format("2006-01-02"))
	}
	if startDate != nil {
		query += " AND (end_date IS NULL OR end_date >= ?)"
		args = append(args, startDate.In(r.loc).Format("2006-01-02"))
	}
	err := r.db.Raw(query, args...).Scan(&results).Error
	return results, err
}

func (r *transactionsRepository) FindCurrentMonthTotalByType(transactionType string) (float64, error) {
	now := time.Now().In(r.loc)
	start, end := monthBounds(now.Year(), now.Month(), r.loc)
	cond, args := r.periodCondition(&start, &end)

	var result struct {
		Total float64 `gorm:"column:total"`
	}
	err := r.db.Model(&model.Transaction{}).
		Select("COALESCE(SUM(amount), 0) as total").
		Where("type = ?", transactionType).
		Where("prepaid_from_id IS NULL").
		Where(cond, args...).
		Scan(&result).Error

	return result.Total, err
}

func (r *transactionsRepository) FindCurrentMonthTotalByTypeAndCategory(transactionType string, categoryID uint) (float64, error) {
	now := time.Now().In(r.loc)
	start, end := monthBounds(now.Year(), now.Month(), r.loc)
	cond, args := r.periodCondition(&start, &end)

	var result struct {
		Total float64 `gorm:"column:total"`
	}
	err := r.db.Model(&model.Transaction{}).
		Select("COALESCE(SUM(amount), 0) as total").
		Where("type = ?", transactionType).
		Where("category_id = ?", categoryID).
		Where("prepaid_from_id IS NULL").
		Where(cond, args...).
		Scan(&result).Error

	return result.Total, err
}

func (r *transactionsRepository) FindNonRecurringMonthlyTotalsByType() ([]MonthlyTypeTotal, error) {
	var results []MonthlyTypeTotal
	tz := r.loc.String()
	err := r.db.Raw(`
		SELECT type,
		       EXTRACT(year FROM (date AT TIME ZONE ?))::int  AS year,
		       EXTRACT(month FROM (date AT TIME ZONE ?))::int AS month,
		       SUM(amount)                                    AS monthly_sum
		FROM transactions
		WHERE is_recurring = false
		  AND date IS NOT NULL
		  AND deleted_at IS NULL
		  AND prepaid_from_id IS NULL
		GROUP BY type, year, month
	`, tz, tz).Scan(&results).Error
	return results, err
}

func (r *transactionsRepository) FindRecurringTransactionSummaryByType() ([]RecurringTypeTransaction, error) {
	var results []RecurringTypeTransaction
	err := r.db.Raw(`
		SELECT type, amount, start_date, end_date
		FROM transactions
		WHERE is_recurring = true
		  AND start_date IS NOT NULL
		  AND deleted_at IS NULL
	`).Scan(&results).Error
	return results, err
}
