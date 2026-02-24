package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthHandler holds dependencies for the health endpoint.
type HealthHandler struct {
	DB *gorm.DB
}

// RegisterHealthRoutes registers health routes on the given router group.
func (h *HealthHandler) RegisterHealthRoutes(rg *gin.RouterGroup) {
	rg.GET("/", h.HealthCheck)
}

// HealthCheck handles GET /health/
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	sqlDB, err := h.DB.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "unhealthy",
			"database": "unhealthy",
		})
		return
	}

	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "unhealthy",
			"database": "unhealthy",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "healthy",
		"database": "healthy",
	})
}
