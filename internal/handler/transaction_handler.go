package handler

import (
	"errors"
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
	locationService    service.LocationService
	loc                *time.Location
}

func NewTransactionHandler(transactionService service.TransactionsService, locationService service.LocationService, loc *time.Location) *TransactionHandler {
	return &TransactionHandler{
		transactionService: transactionService,
		locationService:    locationService,
		loc:                loc,
	}
}

// respondTransactionError maps service validation errors to 400 and
// everything else to 500.
func respondTransactionError(c *gin.Context, err error, message string) {
	status := http.StatusInternalServerError
	if errors.Is(err, service.ErrInvalidTransaction) {
		status = http.StatusBadRequest
	}
	c.JSON(status, gin.H{
		"error":   message,
		"details": err.Error(),
	})
}

// resolveLocation turns an optional free-text location name into a location_id.
// A non-empty name is deduplicated via FindOrCreate; an explicitly empty string
// clears the location. Returns (locationID, cleared, ok); ok is false when an
// error response has already been written.
func (h *TransactionHandler) resolveLocation(c *gin.Context, name *string) (*uint, bool, bool) {
	if name == nil {
		return nil, false, true
	}
	if strings.TrimSpace(*name) == "" {
		return nil, true, true
	}
	location, err := h.locationService.FindOrCreate(c.Request.Context(), *name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to resolve location",
			"details": err.Error(),
		})
		return nil, false, false
	}
	return &location.ID, false, true
}

type CreateTransactionRequest struct {
	IsRecurring   bool  `json:"is_recurring"`
	CategoryID    *uint `json:"category_id,omitempty"`
	SubcategoryID *uint `json:"subcategory_id,omitempty"`
	CreatedById   uint  `json:"created_by_id"`
	Amount      float64 `json:"amount" binding:"required"`
	Type        string  `json:"type" binding:"required"`
	// Deprecated: do not use. Will be removed.
	Subtype     *string `json:"subtype,omitempty"`
	Origin      string  `json:"origin,omitempty"`
	Description *string `json:"description,omitempty"`
	Date        *string `json:"date,omitempty"`
	Frequency   *string `json:"frequency,omitempty"`
	StartDate   *string `json:"start_date,omitempty"`
	EndDate     *string `json:"end_date,omitempty"`
	Location    *string `json:"location,omitempty"`
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

	validOrigins := map[string]bool{"web": true, "api": true, "mcp": true}
	if req.Origin == "" || !validOrigins[req.Origin] {
		req.Origin = "web"
	}

	if req.CategoryID == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "category_id is required",
		})
		return
	}

	transaction := &model.Transaction{
		IsRecurring:   req.IsRecurring,
		CategoryID:    *req.CategoryID,
		SubcategoryID: req.SubcategoryID,
		CreatedById:   req.CreatedById,
		Amount:      req.Amount,
		Type:        req.Type,
		Subtype:     req.Subtype,
		Origin:      req.Origin,
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

	if locationID, _, ok := h.resolveLocation(c, req.Location); ok {
		transaction.LocationID = locationID
	} else {
		return
	}

	if err := h.transactionService.CreateTransaction(c.Request.Context(), transaction); err != nil {
		respondTransactionError(c, err, "Failed to create transaction")
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

	now := time.Now().In(h.loc)
	resp := TransactionDetailResponse{
		Transaction: transaction,
		TotalPaid:   service.ComputeTotalPaid(transaction, now),
		TotalLeft:   service.ComputeTotalLeft(transaction, now),
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

	var transactionType string
	if t := c.Query("type"); t != "" {
		if t != "income" && t != "expense" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid type value",
				"details": "type must be 'income' or 'expense'",
			})
			return
		}
		transactionType = t
	}

	var startDate, endDate *time.Time
	if s := c.Query("start_date"); s != "" {
		parsed, err := time.ParseInLocation("2006-01-02", s, h.loc)
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
		parsed, err := time.ParseInLocation("2006-01-02", s, h.loc)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid end_date format",
				"details": "end_date must be in YYYY-MM-DD format",
			})
			return
		}
		// The query param names an inclusive calendar day; the repository
		// works with half-open intervals.
		exclusive := parsed.AddDate(0, 0, 1)
		endDate = &exclusive
	}

	limit := 50
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || limit <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid limit parameter",
			})
			return
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if _, err := fmt.Sscanf(offsetStr, "%d", &offset); err != nil || offset < 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid offset parameter",
			})
			return
		}
	}

	var transactions []model.Transaction
	var total int64
	var err error

	if currentMonth || len(categoryIDs) > 0 || searchQuery != "" || startDate != nil || endDate != nil || transactionType != "" {
		transactions, total, err = h.transactionService.GetTransactionsWithFilters(c.Request.Context(), currentMonth, categoryIDs, searchQuery, startDate, endDate, transactionType, limit, offset)
	} else {
		transactions, total, err = h.transactionService.GetTransactions(c.Request.Context(), limit, offset)
	}

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
		"total":        total,
		"limit":        limit,
		"offset":       offset,
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
	now := time.Now().In(h.loc)
	month := int(now.Month())
	year := now.Year()

	if monthStr := c.Query("month"); monthStr != "" {
		m, err := strconv.Atoi(monthStr)
		if err != nil || m < 1 || m > 12 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid month: must be between 1 and 12"})
			return
		}
		month = m
	}

	if yearStr := c.Query("year"); yearStr != "" {
		y, err := strconv.Atoi(yearStr)
		if err != nil || y < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid year"})
			return
		}
		year = y
	}

	transactions, err := h.transactionService.GetBiggestTransactions(c.Request.Context(), month, year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch biggest transactions",
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

type UpdateTransactionRequest struct {
	IsRecurring   *bool `json:"is_recurring,omitempty"`
	CategoryID    *uint `json:"category_id,omitempty"`
	SubcategoryID *uint `json:"subcategory_id,omitempty"`
	Amount      *float64 `json:"amount,omitempty"`
	Type        *string  `json:"type,omitempty"`
	// Deprecated: do not use. Will be removed.
	Subtype     *string  `json:"subtype,omitempty"`
	Description *string  `json:"description,omitempty"`
	Date        *string  `json:"date,omitempty"`
	Frequency   *string  `json:"frequency,omitempty"`
	StartDate   *string  `json:"start_date,omitempty"`
	EndDate     *string  `json:"end_date,omitempty"`
	Location    *string  `json:"location,omitempty"`
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
		transaction.CategoryID = *req.CategoryID
	}
	if req.SubcategoryID != nil {
		transaction.SubcategoryID = req.SubcategoryID
	}
	if req.Amount != nil {
		transaction.Amount = *req.Amount
	}
	if req.Type != nil {
		transaction.Type = *req.Type
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

	if locationID, cleared, ok := h.resolveLocation(c, req.Location); ok {
		if cleared {
			transaction.LocationID = nil
		} else if locationID != nil {
			transaction.LocationID = locationID
		}
	} else {
		return
	}

	if err := h.transactionService.UpdateTransaction(c.Request.Context(), transaction); err != nil {
		respondTransactionError(c, err, "Failed to update transaction")
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

	var parsedEndDate time.Time
	var parseErr error
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
		parsedEndDate, parseErr = time.Parse(layout, req.EndDate)
		if parseErr == nil {
			break
		}
	}
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid end_date format",
			"details": "End date must be a valid date string (e.g., 2026-03-01 or 2026-03-01T00:00:00.000Z)",
		})
		return
	}

	transaction.EndDate = &parsedEndDate

	if err := h.transactionService.UpdateTransaction(c.Request.Context(), transaction); err != nil {
		respondTransactionError(c, err, "Failed to end recurring transaction")
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
	startDate, endDate, ok := h.parseDateRangeParams(c)
	if !ok {
		return
	}

	// end_date names an inclusive calendar day; the repository works with
	// half-open intervals.
	transactions, err := h.transactionService.GetTransactionsByDateRange(c.Request.Context(), startDate, endDate.AddDate(0, 0, 1))
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
		"start_date":   c.Query("start_date"),
		"end_date":     c.Query("end_date"),
	})
}

// parseDateRangeParams reads required start_date/end_date (YYYY-MM-DD, in
// the reporting timezone). Returns ok=false with the response written.
func (h *TransactionHandler) parseDateRangeParams(c *gin.Context) (time.Time, time.Time, bool) {
	startStr := c.Query("start_date")
	endStr := c.Query("end_date")
	if startStr == "" || endStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Both start_date and end_date query parameters are required",
		})
		return time.Time{}, time.Time{}, false
	}
	start, err := time.ParseInLocation("2006-01-02", startStr, h.loc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid start_date format",
			"details": "start_date must be in YYYY-MM-DD format",
		})
		return time.Time{}, time.Time{}, false
	}
	end, err := time.ParseInLocation("2006-01-02", endStr, h.loc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid end_date format",
			"details": "end_date must be in YYYY-MM-DD format",
		})
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

// parseMonthYearParams reads optional month/year params, defaulting to the
// current month in the reporting timezone.
func (h *TransactionHandler) parseMonthYearParams(c *gin.Context) (int, int, bool) {
	now := time.Now().In(h.loc)
	month := int(now.Month())
	year := now.Year()

	if monthStr := c.Query("month"); monthStr != "" {
		m, err := strconv.Atoi(monthStr)
		if err != nil || m < 1 || m > 12 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid month: must be between 1 and 12"})
			return 0, 0, false
		}
		month = m
	}
	if yearStr := c.Query("year"); yearStr != "" {
		y, err := strconv.Atoi(yearStr)
		if err != nil || y < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid year"})
			return 0, 0, false
		}
		year = y
	}
	return month, year, true
}

func (h *TransactionHandler) GetMonthlyHistory(c *gin.Context) {
	start, end, ok := h.parseDateRangeParams(c)
	if !ok {
		return
	}

	points, err := h.transactionService.GetMonthlyHistory(c.Request.Context(), start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch monthly history",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, points)
}

func (h *TransactionHandler) GetCategoryHistory(c *gin.Context) {
	start, end, ok := h.parseDateRangeParams(c)
	if !ok {
		return
	}

	var categoryIDs []uint
	if categoryStr := c.Query("category"); categoryStr != "" {
		for _, part := range strings.Split(categoryStr, ",") {
			id, err := strconv.ParseUint(strings.TrimSpace(part), 10, 32)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Invalid category ID",
					"details": fmt.Sprintf("Cannot parse category ID: %s", part),
				})
				return
			}
			categoryIDs = append(categoryIDs, uint(id))
		}
	}

	series, err := h.transactionService.GetCategoryHistory(c.Request.Context(), start, end, categoryIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch category history",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, series)
}

func (h *TransactionHandler) GetMonthOverview(c *gin.Context) {
	month, year, ok := h.parseMonthYearParams(c)
	if !ok {
		return
	}

	overview, err := h.transactionService.GetMonthOverview(c.Request.Context(), month, year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch month overview",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, overview)
}

func (h *TransactionHandler) GetMonthlyExpensesByCategory(c *gin.Context) {
	month, year, ok := h.parseMonthYearParams(c)
	if !ok {
		return
	}

	totals, err := h.transactionService.GetMonthlyExpensesByCategory(c.Request.Context(), month, year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch monthly expenses by category",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"categories": totals,
	})
}

func (h *TransactionHandler) GetAverageByType(c *gin.Context) {
	window := 0
	if w := c.Query("window"); w != "" {
		parsed, err := strconv.Atoi(w)
		if err != nil || parsed < 1 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid window: must be a positive number of months",
			})
			return
		}
		window = parsed
	}

	result, err := h.transactionService.GetAverageByType(c.Request.Context(), window)

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
		parsed, err := time.ParseInLocation("2006-01-02", s, h.loc)
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
		parsed, err := time.ParseInLocation("2006-01-02", s, h.loc)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid end_date format",
				"details": "end_date must be in YYYY-MM-DD format",
			})
			return
		}
		endDate = &parsed
	}

	result, totalIncome, err := h.transactionService.GetAverageByCategory(c.Request.Context(), startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get average by category",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"averageByCategory": result,
		"total_income":      totalIncome,
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
			err.Error() == "transaction was already prepaid" ||
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
