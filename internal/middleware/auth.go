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

// Context keys used to pass data through Gin context.
const (
	ContextUserID = "user_id"
	ContextJTI    = "jti"
)

// TokenType distinguishes access and refresh tokens.
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// Claims represents the JWT claims structure.
// Mirrors Flask-JWT-Extended claims: sub (identity), jti, type, exp.
type Claims struct {
	jwt.RegisteredClaims
	Type TokenType `json:"type"`
}

// ---------- Token Blacklist ----------
// In-memory blacklist equivalent to Flask's jwt_token_blacklist = set().
// Uses sync.RWMutex for concurrent read/write safety.

var (
	blacklist   = make(map[string]struct{})
	blacklistMu sync.RWMutex
)

// AddToBlacklist adds a JTI to the in-memory blacklist (used by logout).
func AddToBlacklist(jti string) {
	blacklistMu.Lock()
	defer blacklistMu.Unlock()
	blacklist[jti] = struct{}{}
}

// IsBlacklisted checks if a JTI has been revoked.
func IsBlacklisted(jti string) bool {
	blacklistMu.RLock()
	defer blacklistMu.RUnlock()
	_, exists := blacklist[jti]
	return exists
}

// ResetBlacklist clears the blacklist (useful for testing).
func ResetBlacklist() {
	blacklistMu.Lock()
	defer blacklistMu.Unlock()
	blacklist = make(map[string]struct{})
}

// ---------- Token Creation ----------

// CreateAccessToken generates a signed JWT access token for the given user ID.
// Identity is stored as a string in the "sub" claim (matching Flask behavior).
func CreateAccessToken(userID int, secret string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			ID:        uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
		Type: AccessToken,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// CreateRefreshToken generates a signed JWT refresh token for the given user ID.
func CreateRefreshToken(userID int, secret string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			ID:        uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
		Type: RefreshToken,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ---------- Middleware ----------

// JWTAuth returns a Gin middleware that validates JWT tokens.
// expectedType specifies whether this middleware expects access or refresh tokens.
// On success, it sets ContextUserID (as int) and ContextJTI in the Gin context.
func JWTAuth(secret string, expectedType TokenType) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header.
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Missing authorization header",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Invalid authorization header format",
			})
			return
		}

		tokenString := parts[1]

		// Parse and validate the token.
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// Ensure signing method is HMAC.
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Invalid or expired token",
			})
			return
		}

		// Verify token type matches expected type.
		if claims.Type != expectedType {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": fmt.Sprintf("Expected %s token", expectedType),
			})
			return
		}

		// Check blacklist (mirrors Flask's check_if_token_revoked callback).
		if IsBlacklisted(claims.ID) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Token has been revoked",
			})
			return
		}

		// Convert identity from string to int (matching Flask's int(get_jwt_identity()) pattern).
		var userID int
		if _, err := fmt.Sscanf(claims.Subject, "%d", &userID); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Invalid token identity",
			})
			return
		}

		// Inject user ID and JTI into context for handlers.
		c.Set(ContextUserID, userID)
		c.Set(ContextJTI, claims.ID)

		c.Next()
	}
}

// GetUserID extracts the authenticated user ID from the Gin context.
// Returns 0 and false if not present.
func GetUserID(c *gin.Context) (int, bool) {
	val, exists := c.Get(ContextUserID)
	if !exists {
		return 0, false
	}
	userID, ok := val.(int)
	return userID, ok
}

// GetJTI extracts the JWT ID from the Gin context.
func GetJTI(c *gin.Context) (string, bool) {
	val, exists := c.Get(ContextJTI)
	if !exists {
		return "", false
	}
	jti, ok := val.(string)
	return jti, ok
}
