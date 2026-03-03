package tests

import (
	"github.com/gin-gonic/gin"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/config"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/db"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/router"
	"gorm.io/gorm"
)

// testConfig returns a Config suitable for integration tests.
func testConfig() *config.Config {
	return &config.Config{
		AppEnv:       "test",
		Port:         "5000",
		DatabaseURL:  "",
		JWTSecretKey: "test-secret-key",
	}
}

// setupTestRouter creates a Gin engine backed by an in-memory SQLite database
// for integration testing. It returns the engine and the GORM DB instance.
func setupTestRouter() (*gin.Engine, *gorm.DB) {
	gin.SetMode(gin.TestMode)

	cfg := testConfig()

	database, err := db.InitTestDB()
	if err != nil {
		panic("failed to initialize test database: " + err.Error())
	}

	r := router.SetupRouter(cfg)
	return r, database
}
