package handler

import (
	"fmt"
	"net/http"

	"github.com/ArthurWerle/transactions/internal/model"
	"github.com/ArthurWerle/transactions/internal/service"
	"github.com/gin-gonic/gin"
)

type SubcategoryHandler struct {
	subcategoryService service.SubcategoryService
}

func NewSubcategoryHandler(subcategoryService service.SubcategoryService) *SubcategoryHandler {
	return &SubcategoryHandler{
		subcategoryService: subcategoryService,
	}
}

type CreateSubcategoryRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
}

func (h *SubcategoryHandler) CreateSubcategory(c *gin.Context) {
	var req CreateSubcategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	subcategory := &model.Subcategory{
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
	}

	if err := h.subcategoryService.CreateSubcategory(c.Request.Context(), subcategory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create subcategory",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Subcategory created successfully",
		"subcategory": subcategory,
	})
}

func (h *SubcategoryHandler) GetSubcategoryByID(c *gin.Context) {
	idParam := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idParam, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid subcategory ID",
		})
		return
	}

	subcategory, err := h.subcategoryService.GetSubcategoryByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Subcategory not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, subcategory)
}

func (h *SubcategoryHandler) GetSubcategories(c *gin.Context) {
	subcategories, err := h.subcategoryService.GetSubcategories(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch subcategories",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subcategories": subcategories,
		"count":         len(subcategories),
	})
}

type UpdateSubcategoryRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Color       *string `json:"color,omitempty"`
}

func (h *SubcategoryHandler) UpdateSubcategory(c *gin.Context) {
	idParam := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idParam, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid subcategory ID",
		})
		return
	}

	subcategory, err := h.subcategoryService.GetSubcategoryByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Subcategory not found",
			"details": err.Error(),
		})
		return
	}

	var req UpdateSubcategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	if req.Name != nil {
		subcategory.Name = *req.Name
	}
	if req.Description != nil {
		subcategory.Description = *req.Description
	}
	if req.Color != nil {
		subcategory.Color = *req.Color
	}

	if err := h.subcategoryService.UpdateSubcategory(c.Request.Context(), subcategory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update subcategory",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Subcategory updated successfully",
		"subcategory": subcategory,
	})
}

func (h *SubcategoryHandler) DeleteSubcategory(c *gin.Context) {
	idParam := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idParam, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid subcategory ID",
		})
		return
	}

	if err := h.subcategoryService.DeleteSubcategory(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete subcategory",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Subcategory deleted successfully",
	})
}
