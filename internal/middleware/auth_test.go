package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/flask-rest-api-jwt-guy/internal/config"
	"github.com/flask-rest-api-jwt-guy/internal/middleware"
)

func testConfig() *config.Config {
	return &config.Config{
		JWTSecret:       "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 30 * 24 * time.Hour,
	}
}

func TestGenerateAndValidateAccessToken(t *testing.T) {
	cfg := testConfig()
	blacklist := middleware.NewTokenBlacklist()

	token, err := middleware.GenerateAccessToken(42, cfg)
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}

	// Set up a test router with auth middleware
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test", middleware.AuthMiddleware(cfg, blacklist), func(c *gin.Context) {
		userID, err := middleware.GetUserID(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRefreshTokenRejectedByAccessMiddleware(t *testing.T) {
	cfg := testConfig()
	blacklist := middleware.NewTokenBlacklist()

	refreshToken, err := middleware.GenerateRefreshToken(42, cfg)
	if err != nil {
		t.Fatalf("Failed to generate refresh token: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test", middleware.AuthMiddleware(cfg, blacklist), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+refreshToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 when using refresh token with access middleware, got %d", w.Code)
	}
}

func TestBlacklistedTokenRejected(t *testing.T) {
	cfg := testConfig()
	blacklist := middleware.NewTokenBlacklist()

	token, err := middleware.GenerateAccessToken(42, cfg)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Use the token first to extract JTI (simulate a request to extract it)
	gin.SetMode(gin.TestMode)
	var jti string
	router := gin.New()
	router.GET("/test", middleware.AuthMiddleware(cfg, blacklist), func(c *gin.Context) {
		j, _ := c.Get(middleware.ContextJTI)
		jti = j.(string)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 before blacklist, got %d", w.Code)
	}

	// Blacklist the JTI
	blacklist.Add(jti)

	// Now the same token should be rejected
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for blacklisted token, got %d", w2.Code)
	}
}

func TestMissingAuthorizationHeader(t *testing.T) {
	cfg := testConfig()
	blacklist := middleware.NewTokenBlacklist()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test", middleware.AuthMiddleware(cfg, blacklist), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for missing auth header, got %d", w.Code)
	}
}

func TestInvalidTokenFormat(t *testing.T) {
	cfg := testConfig()
	blacklist := middleware.NewTokenBlacklist()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test", middleware.AuthMiddleware(cfg, blacklist), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for invalid auth format, got %d", w.Code)
	}
}
