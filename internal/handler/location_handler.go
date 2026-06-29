package handler

import (
	"fmt"
	"net/http"

	"github.com/ArthurWerle/transactions/internal/service"
	"github.com/gin-gonic/gin"
)

type LocationHandler struct {
	locationService service.LocationService
}

func NewLocationHandler(locationService service.LocationService) *LocationHandler {
	return &LocationHandler{
		locationService: locationService,
	}
}

type CreateLocationRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *LocationHandler) CreateLocation(c *gin.Context) {
	var req CreateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	location, err := h.locationService.FindOrCreate(c.Request.Context(), req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create location",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Location created successfully",
		"location": location,
	})
}

func (h *LocationHandler) GetLocationByID(c *gin.Context) {
	idParam := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idParam, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid location ID",
		})
		return
	}

	location, err := h.locationService.GetLocationByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Location not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, location)
}

func (h *LocationHandler) GetLocations(c *gin.Context) {
	locations, err := h.locationService.GetLocations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch locations",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"locations": locations,
		"count":     len(locations),
	})
}

type UpdateLocationRequest struct {
	Name *string `json:"name,omitempty"`
}

func (h *LocationHandler) UpdateLocation(c *gin.Context) {
	idParam := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idParam, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid location ID",
		})
		return
	}

	location, err := h.locationService.GetLocationByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Location not found",
			"details": err.Error(),
		})
		return
	}

	var req UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	if req.Name != nil {
		location.Name = *req.Name
	}

	if err := h.locationService.UpdateLocation(c.Request.Context(), location); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update location",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Location updated successfully",
		"location": location,
	})
}

func (h *LocationHandler) DeleteLocation(c *gin.Context) {
	idParam := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idParam, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid location ID",
		})
		return
	}

	if err := h.locationService.DeleteLocation(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete location",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Location deleted successfully",
	})
}

type MergeLocationsRequest struct {
	SourceID uint `json:"source_id" binding:"required"`
	TargetID uint `json:"target_id" binding:"required"`
}

func (h *LocationHandler) MergeLocations(c *gin.Context) {
	var req MergeLocationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	if err := h.locationService.MergeLocations(c.Request.Context(), req.SourceID, req.TargetID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to merge locations",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Locations merged successfully",
	})
}
