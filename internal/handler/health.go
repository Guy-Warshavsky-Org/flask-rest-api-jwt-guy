package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/db"
)

// RegisterHealthRoutes registers the health check route group.
func RegisterHealthRoutes(rg *gin.RouterGroup) {
	health := rg.Group("/health")
	{
		health.GET("/", HealthCheck)
	}
}

// HealthCheck performs a lightweight liveness/readiness probe.
// Returns 200 when the API process is up and the database is reachable.
// Returns 503 if the database connection fails.
func HealthCheck(c *gin.Context) {
	sqlDB, err := db.DB.DB()
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
