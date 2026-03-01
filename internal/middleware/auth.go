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
)

// TokenBlacklist provides a concurrency-safe in-memory store for revoked JTIs.
// This mirrors the Flask app's in-memory set() for token blacklisting.
type TokenBlacklist struct {
	mu    sync.RWMutex
	jtis  map[string]struct{}
}

// NewTokenBlacklist creates a new empty token blacklist.
func NewTokenBlacklist() *TokenBlacklist {
	return &TokenBlacklist{
		jtis: make(map[string]struct{}),
	}
}

// Add adds a JTI to the blacklist (revokes a token).
func (b *TokenBlacklist) Add(jti string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.jtis[jti] = struct{}{}
}

// Contains checks whether a JTI is in the blacklist.
func (b *TokenBlacklist) Contains(jti string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, ok := b.jtis[jti]
	return ok
}

// Claims defines the JWT claims structure used by the application.
type Claims struct {
	jwt.RegisteredClaims
	TokenType string `json:"type"` // "access" or "refresh"
}

// Token expiration durations matching Flask config:
// JWT_ACCESS_TOKEN_EXPIRES = timedelta(minutes=15)
// JWT_REFRESH_TOKEN_EXPIRES = timedelta(days=30)
const (
	AccessTokenExpiry  = 15 * time.Minute
	RefreshTokenExpiry = 30 * 24 * time.Hour
)

// GenerateAccessToken creates a new JWT access token for the given user ID.
// The user ID is stored as a string in the "sub" claim, matching Flask-JWT-Extended behavior.
func GenerateAccessToken(userID uint, secretKey string) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		TokenType: "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

// GenerateRefreshToken creates a new JWT refresh token for the given user ID.
func GenerateRefreshToken(userID uint, secretKey string) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			ExpiresAt: jwt.NewNumericDate(now.Add(RefreshTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		TokenType: "refresh",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

// parseToken parses and validates a JWT token string, returning claims on success.
func parseToken(tokenString, secretKey string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// extractBearerToken extracts the token string from the Authorization header.
func extractBearerToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// JWTAuth returns Gin middleware that validates access tokens.
// It checks the Authorization Bearer header, verifies the token signature and expiry,
// checks the token against the blacklist, and sets the user ID in the Gin context.
func JWTAuth(secretKey string, blacklist *TokenBlacklist) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractBearerToken(c)
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"msg": "Missing Authorization Header",
			})
			return
		}

		claims, err := parseToken(tokenString, secretKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"msg": "Token has expired",
			})
			return
		}

		// Check token type - this middleware expects access tokens
		if claims.TokenType != "access" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"msg": "Only non-refresh tokens are allowed",
			})
			return
		}

		// Check blacklist
		if blacklist.Contains(claims.ID) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"msg": "Token has been revoked",
			})
			return
		}

		// Set user identity and JWT claims in context
		c.Set("jwt_identity", claims.Subject)
		c.Set("jwt_claims", claims)
		c.Set("jwt_jti", claims.ID)

		c.Next()
	}
}

// JWTRefresh returns Gin middleware that validates refresh tokens.
// Similar to JWTAuth but expects refresh token type.
func JWTRefresh(secretKey string, blacklist *TokenBlacklist) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractBearerToken(c)
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"msg": "Missing Authorization Header",
			})
			return
		}

		claims, err := parseToken(tokenString, secretKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"msg": "Token has expired",
			})
			return
		}

		// Check token type - this middleware expects refresh tokens
		if claims.TokenType != "refresh" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"msg": "Only refresh tokens are allowed",
			})
			return
		}

		// Check blacklist
		if blacklist.Contains(claims.ID) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"msg": "Token has been revoked",
			})
			return
		}

		// Set user identity and JWT claims in context
		c.Set("jwt_identity", claims.Subject)
		c.Set("jwt_claims", claims)
		c.Set("jwt_jti", claims.ID)

		c.Next()
	}
}

// GetIdentity extracts the user identity string from the Gin context.
// Returns empty string if not set.
func GetIdentity(c *gin.Context) string {
	val, exists := c.Get("jwt_identity")
	if !exists {
		return ""
	}
	identity, ok := val.(string)
	if !ok {
		return ""
	}
	return identity
}

// GetJTI extracts the JWT ID from the Gin context.
func GetJTI(c *gin.Context) string {
	val, exists := c.Get("jwt_jti")
	if !exists {
		return ""
	}
	jti, ok := val.(string)
	if !ok {
		return ""
	}
	return jti
}
