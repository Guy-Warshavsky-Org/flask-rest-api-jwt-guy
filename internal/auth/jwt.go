package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenType distinguishes access tokens from refresh tokens,
// mirroring Flask-JWT-Extended's "type" claim ("access" or "refresh").
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// Claims represents the JWT claims used by this application.
// It embeds jwt.RegisteredClaims for standard fields (sub, jti, exp, iat, nbf)
// and adds a Type field to distinguish access from refresh tokens,
// matching Flask-JWT-Extended's token structure.
type Claims struct {
	jwt.RegisteredClaims
	Type string `json:"type"`
}

// GenerateAccessToken creates a new signed JWT access token for the given user ID.
// The identity (user ID) is stored in the "sub" claim as a string,
// matching Flask-JWT-Extended's identity=str(user.id) convention.
func GenerateAccessToken(userID string, secretKey string, expiry time.Duration) (string, error) {
	return generateToken(userID, secretKey, expiry, TokenTypeAccess)
}

// GenerateRefreshToken creates a new signed JWT refresh token for the given user ID.
func GenerateRefreshToken(userID string, secretKey string, expiry time.Duration) (string, error) {
	return generateToken(userID, secretKey, expiry, TokenTypeRefresh)
}

// generateToken creates a signed JWT with the specified claims.
func generateToken(userID string, secretKey string, expiry time.Duration, tokenType string) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        uuid.New().String(), // JTI for blacklist/revocation
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
		Type: tokenType,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

// ParseToken parses and validates a JWT token string, returning the claims.
// It verifies the signature using the provided secret key and checks
// that the token is not expired.
func ParseToken(tokenString string, secretKey string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Ensure the signing method is HMAC (HS256)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
