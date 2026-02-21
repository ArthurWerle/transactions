package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ArthurWerle/transactions/internal/model"
	"github.com/ArthurWerle/transactions/internal/service"
	"github.com/gin-gonic/gin"
)

type TransactionHandler struct {
	transactionService service.TransactionsService
}

func NewTransactionHandler(transactionService service.TransactionsService) *TransactionHandler {
	return &TransactionHandler{
		transactionService: transactionService,
	}
}

type CreateTransactionRequest struct {
	IsRecurring bool    `json:"is_recurring"`
	CategoryID  *uint   `json:"category_id,omitempty"`
	CreatedById uint    `json:"created_by_id"`
	Amount      float64 `json:"amount" binding:"required"`
	Type        string  `json:"type" binding:"required"`
	Subtype     *string `json:"subtype,omitempty"`
	Description *string `json:"description,omitempty"`
	Date        *string `json:"date,omitempty"`
	Frequency   *string `json:"frequency,omitempty"`
	StartDate   *string `json:"start_date,omitempty"`
	EndDate     *string `json:"end_date,omitempty"`
}

func (h *TransactionHandler) CreateTransaction(c *gin.Context) {
	var req CreateTransactionRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate amount
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Amount must be greater than 0",
		})
		return
	}

	// Validate transaction type
	validTypes := map[string]bool{
		"income":  true,
		"expense": true,
	}
	if !validTypes[req.Type] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Type must be either 'income' or 'expense'",
		})
		return
	}

	transaction := &model.Transaction{
		IsRecurring: req.IsRecurring,
		CategoryID:  req.CategoryID,
		CreatedById: req.CreatedById,
		Amount:      req.Amount,
		Type:        req.Type,
		Subtype:     req.Subtype,
		Description: req.Description,
		Frequency:   req.Frequency,
	}

	// Parse dates if provided
	if req.Date != nil {
		parsedDate, err := time.Parse(time.RFC3339, *req.Date)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid date format",
				"details": "Date must be in RFC3339 format (e.g., 2024-01-15T10:30:00Z)",
			})
			return
		}
		transaction.Date = &parsedDate
	}
	if req.StartDate != nil {
		parsedStartDate, err := time.Parse("2006-01-02", *req.StartDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid start_date format",
				"details": "Start date must be in YYYY-MM-DD format (e.g., 2024-01-15)",
			})
			return
		}
		transaction.StartDate = &parsedStartDate
	}
	if req.EndDate != nil {
		parsedEndDate, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid end_date format",
				"details": "End date must be in YYYY-MM-DD format (e.g., 2024-12-31)",
			})
			return
		}
		transaction.EndDate = &parsedEndDate
	}

	if err := h.transactionService.CreateTransaction(c.Request.Context(), transaction); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create transaction",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Transaction created successfully",
		"transaction": transaction,
	})
}

type TransactionDetailResponse struct {
	*model.Transaction
	TotalPaid            *float64 `json:"total_paid,omitempty"`
	TotalLeft            *float64 `json:"total_left,omitempty"`
	CategoryMonthPercent *float64 `json:"category_month_percent,omitempty"`
	TotalMonthPercent    *float64 `json:"total_month_percent,omitempty"`
}

func computeTotalPaid(tx *model.Transaction) *float64 {
	if !tx.IsRecurring || tx.StartDate == nil {
		return nil
	}

	now := time.Now()
	startDate := time.Date(tx.StartDate.Year(), tx.StartDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	effectiveEnd := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	if tx.EndDate != nil {
		endDate := time.Date(tx.EndDate.Year(), tx.EndDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		if endDate.Before(effectiveEnd) {
			effectiveEnd = endDate
		}
	}

	if effectiveEnd.Before(startDate) {
		return nil
	}

	years := effectiveEnd.Year() - startDate.Year()
	months := int(effectiveEnd.Month()) - int(startDate.Month())
	monthsPaid := years*12 + months

	total := tx.Amount * float64(monthsPaid)
	return &total
}

func computeTotalLeft(tx *model.Transaction) *float64 {
	if !tx.IsRecurring || tx.StartDate == nil || tx.EndDate == nil {
		return nil
	}

	now := time.Now()
	effectiveStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(tx.EndDate.Year(), tx.EndDate.Month(), 1, 0, 0, 0, 0, time.UTC)

	if endDate.Before(effectiveStart) {
		zero := 0.0
		return &zero
	}

	years := endDate.Year() - effectiveStart.Year()
	months := int(endDate.Month()) - int(effectiveStart.Month())
	monthsLeft := years*12 + months

	total := tx.Amount * float64(monthsLeft)
	return &total
}

func (h *TransactionHandler) GetTransactionByID(c *gin.Context) {
	idParam := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idParam, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid transaction ID",
		})
		return
	}

	transaction, err := h.transactionService.GetTransactionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Transaction not found",
			"details": err.Error(),
		})
		return
	}

	percentages, err := h.transactionService.GetTransactionMonthlyPercentages(c.Request.Context(), transaction)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to compute transaction percentages",
			"details": err.Error(),
		})
		return
	}

	resp := TransactionDetailResponse{
		Transaction: transaction,
		TotalPaid:   computeTotalPaid(transaction),
		TotalLeft:   computeTotalLeft(transaction),
	}
	if percentages != nil {
		resp.CategoryMonthPercent = percentages.CategoryMonthPercent
		resp.TotalMonthPercent = percentages.TotalMonthPercent
	}

	c.JSON(http.StatusOK, resp)
}

func (h *TransactionHandler) GetTransactions(c *gin.Context) {
	currentMonth := false
	if currentMonthStr := c.Query("currentMonth"); currentMonthStr == "true" {
		currentMonth = true
	}

	var categoryIDs []uint
	if categoryStr := c.Query("category"); categoryStr != "" {
		categoryParts := strings.Split(categoryStr, ",")
		for _, part := range categoryParts {
			categoryID, err := strconv.ParseUint(strings.TrimSpace(part), 10, 32)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Invalid category ID",
					"details": fmt.Sprintf("Cannot parse category ID: %s", part),
				})
				return
			}
			categoryIDs = append(categoryIDs, uint(categoryID))
		}
	}

	var searchQuery string
	if queryStr := c.Query("query"); queryStr != "" {
		searchQuery = queryStr
	}

	var startDate, endDate *time.Time
	if s := c.Query("start_date"); s != "" {
		parsed, err := time.Parse("2006-01-02", s)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid start_date format",
				"details": "start_date must be in YYYY-MM-DD format",
			})
			return
		}
		startDate = &parsed
	}
	if s := c.Query("end_date"); s != "" {
		parsed, err := time.Parse("2006-01-02", s)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid end_date format",
				"details": "end_date must be in YYYY-MM-DD format",
			})
			return
		}
		endDate = &parsed
	}

	var transactions []model.Transaction
	var err error

	if currentMonth || len(categoryIDs) > 0 || searchQuery != "" || startDate != nil || endDate != nil {
		transactions, err = h.transactionService.GetTransactionsWithFilters(c.Request.Context(), currentMonth, categoryIDs, searchQuery, startDate, endDate)
	} else {
		transactions, err = h.transactionService.GetTransactions(c.Request.Context())
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch transactions",
			"details": err.Error(),
		})
		return
	}

	var sum float64
	for _, t := range transactions {
		sum += t.Amount
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"count":        len(transactions),
		"sum":          sum,
	})
}

func (h *TransactionHandler) GetLatestTransactions(c *gin.Context) {
	var transactions []model.Transaction
	var err error

	transactions, err = h.transactionService.GetLatestTransactions(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch latest transactions",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"count":        len(transactions),
	})
}

func (h *TransactionHandler) GetBiggestTransactions(c *gin.Context) {
	var transactions []model.Transaction
	var err error

	transactions, err = h.transactionService.GetBiggestTransactions(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch biggest transactions",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"count":        len(transactions),
	})
}

type UpdateTransactionRequest struct {
	IsRecurring *bool    `json:"is_recurring,omitempty"`
	CategoryID  *uint    `json:"category_id,omitempty"`
	Amount      *float64 `json:"amount,omitempty"`
	Type        *string  `json:"type,omitempty"`
	Subtype     *string  `json:"subtype,omitempty"`
	Description *string  `json:"description,omitempty"`
	Date        *string  `json:"date,omitempty"`
	Frequency   *string  `json:"frequency,omitempty"`
	StartDate   *string  `json:"start_date,omitempty"`
	EndDate     *string  `json:"end_date,omitempty"`
}

func (h *TransactionHandler) UpdateTransaction(c *gin.Context) {
	idParam := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idParam, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid transaction ID",
		})
		return
	}

	// Get existing transaction
	transaction, err := h.transactionService.GetTransactionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Transaction not found",
			"details": err.Error(),
		})
		return
	}

	var req UpdateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Update fields if provided
	if req.IsRecurring != nil {
		transaction.IsRecurring = *req.IsRecurring
	}
	if req.CategoryID != nil {
		transaction.CategoryID = req.CategoryID
	}
	if req.Amount != nil {
		if *req.Amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Amount must be greater than 0",
			})
			return
		}
		transaction.Amount = *req.Amount
	}
	if req.Type != nil {
		validTypes := map[string]bool{
			"income":  true,
			"expense": true,
		}
		if !validTypes[*req.Type] {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Type must be either 'income' or 'expense'",
			})
			return
		}
		transaction.Type = *req.Type
	}
	if req.Subtype != nil {
		transaction.Subtype = req.Subtype
	}
	if req.Description != nil {
		transaction.Description = req.Description
	}
	if req.Frequency != nil {
		transaction.Frequency = req.Frequency
	}

	// Parse dates if provided
	if req.Date != nil {
		parsedDate, err := time.Parse(time.RFC3339, *req.Date)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid date format",
				"details": "Date must be in RFC3339 format (e.g., 2024-01-15T10:30:00Z)",
			})
			return
		}
		transaction.Date = &parsedDate
	}
	if req.StartDate != nil {
		parsedStartDate, err := time.Parse("2006-01-02", *req.StartDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid start_date format",
				"details": "Start date must be in YYYY-MM-DD format (e.g., 2024-01-15)",
			})
			return
		}
		transaction.StartDate = &parsedStartDate
	}
	if req.EndDate != nil {
		parsedEndDate, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid end_date format",
				"details": "End date must be in YYYY-MM-DD format (e.g., 2024-12-31)",
			})
			return
		}
		transaction.EndDate = &parsedEndDate
	}

	if err := h.transactionService.UpdateTransaction(c.Request.Context(), transaction); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update transaction",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Transaction updated successfully",
		"transaction": transaction,
	})
}

type EndRecurringTransactionRequest struct {
	EndDate string `json:"end_date" binding:"required"`
}

func (h *TransactionHandler) EndRecurringTransaction(c *gin.Context) {
	idParam := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idParam, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid transaction ID",
		})
		return
	}

	transaction, err := h.transactionService.GetTransactionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Transaction not found",
			"details": err.Error(),
		})
		return
	}

	if !transaction.IsRecurring {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Transaction is not recurring",
		})
		return
	}

	var req EndRecurringTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": "end_date is required in YYYY-MM-DD format (e.g., 2026-03-01)",
		})
		return
	}

	parsedEndDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid end_date format",
			"details": "End date must be in YYYY-MM-DD format (e.g., 2026-03-01)",
		})
		return
	}

	transaction.EndDate = &parsedEndDate

	if err := h.transactionService.UpdateTransaction(c.Request.Context(), transaction); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to end recurring transaction",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Recurring transaction ended successfully",
		"transaction": transaction,
	})
}

func (h *TransactionHandler) DeleteTransaction(c *gin.Context) {
	idParam := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idParam, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid transaction ID",
		})
		return
	}

	if err := h.transactionService.DeleteTransaction(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete transaction",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Transaction deleted successfully",
	})
}

func (h *TransactionHandler) GetTransactionsByDateRange(c *gin.Context) {
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Both start_date and end_date query parameters are required",
		})
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid start_date format",
			"details": "Start date must be in YYYY-MM-DD format (e.g., 2024-01-15)",
		})
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid end_date format",
			"details": "End date must be in YYYY-MM-DD format (e.g., 2024-12-31)",
		})
		return
	}

	transactions, err := h.transactionService.GetTransactionsByDateRange(c.Request.Context(), startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch transactions",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"count":        len(transactions),
		"start_date":   startDateStr,
		"end_date":     endDateStr,
	})
}

func (h *TransactionHandler) GetTransactionsByType(c *gin.Context) {
	transactionType := c.Query("type")
	if transactionType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Type query parameter is required",
		})
		return
	}

	// Validate type
	validTypes := map[string]bool{
		"income":  true,
		"expense": true,
	}
	if !validTypes[transactionType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Type must be either 'income' or 'expense'",
		})
		return
	}

	// Parse pagination parameters
	limit := 50 // default
	offset := 0 // default

	if limitStr := c.Query("limit"); limitStr != "" {
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid limit parameter",
			})
			return
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if _, err := fmt.Sscanf(offsetStr, "%d", &offset); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid offset parameter",
			})
			return
		}
	}

	transactions, err := h.transactionService.GetTransactionsByType(c.Request.Context(), transactionType, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch transactions",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"count":        len(transactions),
		"type":         transactionType,
		"limit":        limit,
		"offset":       offset,
	})
}

func (h *TransactionHandler) GetRecurringTransactions(c *gin.Context) {
	transactions, err := h.transactionService.GetRecurringTransactions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch recurring transactions",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"count":        len(transactions),
	})
}

func (h *TransactionHandler) GetTransactionsByCategory(c *gin.Context) {
	categoryIDParam := c.Param("categoryId")
	var categoryID uint
	if _, err := fmt.Sscanf(categoryIDParam, "%d", &categoryID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid category ID",
		})
		return
	}

	// Parse pagination parameters
	limit := 50 // default
	offset := 0 // default

	if limitStr := c.Query("limit"); limitStr != "" {
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid limit parameter",
			})
			return
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if _, err := fmt.Sscanf(offsetStr, "%d", &offset); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid offset parameter",
			})
			return
		}
	}

	transactions, err := h.transactionService.GetTransactionsByCategory(c.Request.Context(), categoryID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch transactions",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"count":        len(transactions),
		"category_id":  categoryID,
		"limit":        limit,
		"offset":       offset,
	})
}

type GetTransactionsByCategoriesRequest struct {
	CategoryIDs []uint `json:"category_ids" binding:"required"`
	Limit       int    `json:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty"`
}

func (h *TransactionHandler) GetTransactionsByCategories(c *gin.Context) {
	var req GetTransactionsByCategoriesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Set defaults
	limit := req.Limit
	if limit == 0 {
		limit = 50
	}
	offset := req.Offset

	transactions, err := h.transactionService.GetTransactionsByCategories(c.Request.Context(), req.CategoryIDs, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch transactions",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"count":        len(transactions),
		"category_ids": req.CategoryIDs,
		"limit":        limit,
		"offset":       offset,
	})
}

func (h *TransactionHandler) GetAverageByType(c *gin.Context) {
	result, err := h.transactionService.GetAverageByType(c.Request.Context())

	if err != nil {
		status := http.StatusInternalServerError

		c.JSON(status, gin.H{
			"error":   "Failed to get average by type",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"averageByType": result,
	})
}

func (h *TransactionHandler) GetAverageByCategory(c *gin.Context) {
	var startDate, endDate *time.Time

	if s := c.Query("start_date"); s != "" {
		parsed, err := time.Parse("2006-01-02", s)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid start_date format",
				"details": "start_date must be in YYYY-MM-DD format",
			})
			return
		}
		startDate = &parsed
	}

	if s := c.Query("end_date"); s != "" {
		parsed, err := time.Parse("2006-01-02", s)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid end_date format",
				"details": "end_date must be in YYYY-MM-DD format",
			})
			return
		}
		endDate = &parsed
	}

	result, err := h.transactionService.GetAverageByCategory(c.Request.Context(), startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get average by category",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"averageByCategory": result,
	})
}

func (h *TransactionHandler) PrepayTransaction(c *gin.Context) {
	idParam := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idParam, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid transaction ID",
		})
		return
	}

	result, err := h.transactionService.PrepayTransaction(c.Request.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "transaction is not recurring" ||
			err.Error() == "recurring transaction has no end_date defined" ||
			err.Error() == "recurring transaction has no start_date defined" ||
			err.Error() == "recurring transaction has already ended" ||
			err.Error() == "no remaining installments to prepay" {
			status = http.StatusBadRequest
		}
		if err.Error() == "record not found" {
			status = http.StatusNotFound
		}

		c.JSON(status, gin.H{
			"error":   "Failed to prepay transaction",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":              "Transaction prepaid successfully",
		"original_transaction": result.OriginalTransaction,
		"prepay_transaction":   result.PrepayTransaction,
		"remaining_months":     result.RemainingMonths,
		"prepaid_amount":       result.PrepaidAmount,
	})
}
