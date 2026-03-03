package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/auth"
)

// ContextKeyUserID is the key used to store the authenticated user's identity
// in the Gin context. Handlers retrieve it via c.GetString("user_id").
const ContextKeyUserID = "user_id"

// ContextKeyJTI is the key used to store the token's JTI in the Gin context.
// The logout handler uses this to add the JTI to the blacklist.
const ContextKeyJTI = "jwt_jti"

// RequireAccessToken returns a Gin middleware that validates JWT access tokens.
// It extracts the Bearer token from the Authorization header, parses and validates
// it, checks that the token type is "access", verifies the JTI is not blacklisted,
// and injects the user identity into the Gin context.
//
// This mirrors Flask-JWT-Extended's @jwt_required() decorator.
func RequireAccessToken(secretKey string) gin.HandlerFunc {
	return requireToken(secretKey, auth.TokenTypeAccess)
}

// RequireRefreshToken returns a Gin middleware that validates JWT refresh tokens.
// It works identically to RequireAccessToken but expects the token type to be "refresh".
//
// This mirrors Flask-JWT-Extended's @jwt_required(refresh=True) decorator.
func RequireRefreshToken(secretKey string) gin.HandlerFunc {
	return requireToken(secretKey, auth.TokenTypeRefresh)
}

// requireToken is the shared implementation for both access and refresh token middleware.
func requireToken(secretKey string, expectedType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"msg": "Missing Authorization Header",
			})
			return
		}

		// Expect "Bearer <token>" format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
				"msg": "Bad Authorization header. Expected 'Bearer <JWT>'",
			})
			return
		}

		tokenString := parts[1]

		// Parse and validate the token
		claims, err := auth.ParseToken(tokenString, secretKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
				"msg": "Not enough segments",
			})
			return
		}

		// Verify token type matches expected type
		if claims.Type != expectedType {
			c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
				"msg": "Only " + expectedType + " tokens are allowed",
			})
			return
		}

		// Check if the token has been revoked (blacklisted)
		if auth.TokenBlacklist.Contains(claims.ID) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"msg": "Token has been revoked",
			})
			return
		}

		// Set user identity and JTI in Gin context for downstream handlers
		c.Set(ContextKeyUserID, claims.Subject)
		c.Set(ContextKeyJTI, claims.ID)

		c.Next()
	}
}
