package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/flask-rest-api-jwt-guy/internal/config"
)

// contextKey constants for injecting user identity into Gin context.
const (
	ContextUserID   = "userID"
	ContextJTI      = "jti"
	ContextTokenType = "tokenType"
)

// TokenBlacklist provides thread-safe in-memory token blacklist (mirrors Flask's in-memory set).
type TokenBlacklist struct {
	mu    sync.RWMutex
	jtiSet map[string]struct{}
}

// NewTokenBlacklist creates a new empty blacklist.
func NewTokenBlacklist() *TokenBlacklist {
	return &TokenBlacklist{
		jtiSet: make(map[string]struct{}),
	}
}

// Add adds a JTI to the blacklist.
func (b *TokenBlacklist) Add(jti string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.jtiSet[jti] = struct{}{}
}

// Contains checks if a JTI is blacklisted.
func (b *TokenBlacklist) Contains(jti string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, ok := b.jtiSet[jti]
	return ok
}

// Claims represents the JWT claims for access and refresh tokens.
type Claims struct {
	Sub  string `json:"sub"`
	Type string `json:"type"` // "access" or "refresh"
	JTI  string `json:"jti"`
	jwt.RegisteredClaims
}

// GenerateAccessToken creates a new JWT access token for the given user ID.
func GenerateAccessToken(userID uint, cfg *config.Config) (string, error) {
	now := time.Now()
	claims := Claims{
		Sub:  fmt.Sprintf("%d", userID),
		Type: "access",
		JTI:  uuid.New().String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(cfg.AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

// GenerateRefreshToken creates a new JWT refresh token for the given user ID.
func GenerateRefreshToken(userID uint, cfg *config.Config) (string, error) {
	now := time.Now()
	claims := Claims{
		Sub:  fmt.Sprintf("%d", userID),
		Type: "refresh",
		JTI:  uuid.New().String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(cfg.RefreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

// parseToken parses and validates a JWT token string.
func parseToken(tokenString string, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// AuthMiddleware returns a Gin middleware that validates JWT access tokens.
func AuthMiddleware(cfg *config.Config, blacklist *TokenBlacklist) gin.HandlerFunc {
	return authMiddleware(cfg, blacklist, "access")
}

// RefreshMiddleware returns a Gin middleware that validates JWT refresh tokens.
func RefreshMiddleware(cfg *config.Config, blacklist *TokenBlacklist) gin.HandlerFunc {
	return authMiddleware(cfg, blacklist, "refresh")
}

func authMiddleware(cfg *config.Config, blacklist *TokenBlacklist, expectedType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Missing authorization header"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid authorization header"})
			c.Abort()
			return
		}

		claims, err := parseToken(parts[1], cfg.JWTSecret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Check token type
		if claims.Type != expectedType {
			c.JSON(http.StatusUnauthorized, gin.H{"message": fmt.Sprintf("Expected %s token", expectedType)})
			c.Abort()
			return
		}

		// Check blacklist
		if blacklist.Contains(claims.JTI) {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Token has been revoked"})
			c.Abort()
			return
		}

		// Set user identity in context
		c.Set(ContextUserID, claims.Sub)
		c.Set(ContextJTI, claims.JTI)
		c.Set(ContextTokenType, claims.Type)

		c.Next()
	}
}

// GetUserID extracts the authenticated user ID from Gin context as a uint.
func GetUserID(c *gin.Context) (uint, error) {
	idStr, exists := c.Get(ContextUserID)
	if !exists {
		return 0, fmt.Errorf("user ID not found in context")
	}

	var userID uint
	if _, err := fmt.Sscanf(idStr.(string), "%d", &userID); err != nil {
		return 0, fmt.Errorf("invalid user ID: %w", err)
	}
	return userID, nil
}
