package handler

import (
	"net/http"
	"strconv"

	"flask-rest-api-jwt-guy/internal/middleware"
	"flask-rest-api-jwt-guy/internal/model"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// userRequest holds the JSON body for registration and login.
type userRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// userResponse matches the Flask UserSchema output (excludes password_hash).
type userResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

func toUserResponse(u *model.User) userResponse {
	return userResponse{
		ID:       u.ID,
		Username: u.Username,
	}
}

// RegisterUser handles POST /user/register
// Matches Flask: returns user JSON with 201, or {"message": "User exists"} with 400.
func RegisterUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req userRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
			return
		}

		// Check if user already exists
		var existing model.User
		if err := db.Where("username = ?", req.Username).First(&existing).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "User exists"})
			return
		}

		// Hash password using bcrypt
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error creating user"})
			return
		}

		user := model.User{
			Username:     req.Username,
			PasswordHash: string(hash),
		}

		if err := db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error creating user"})
			return
		}

		c.JSON(http.StatusCreated, toUserResponse(&user))
	}
}

// Login handles POST /user/login
// Matches Flask: returns {access_token, refresh_token} with 200,
// or {"message": "Invalid credentials"} with 401.
func Login(db *gorm.DB, secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req userRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
			return
		}

		var user model.User
		if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid credentials"})
			return
		}

		// Check password
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid credentials"})
			return
		}

		accessToken, err := middleware.GenerateAccessToken(user.ID, secretKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error generating token"})
			return
		}

		refreshToken, err := middleware.GenerateRefreshToken(user.ID, secretKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error generating token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		})
	}
}

// Logout handles POST /user/logout
// Matches Flask: blacklists the current access token's JTI and returns {"message": "Logged out"}.
func Logout(blacklist *middleware.TokenBlacklist) gin.HandlerFunc {
	return func(c *gin.Context) {
		jti := middleware.GetJTI(c)
		if jti != "" {
			blacklist.Add(jti)
		}

		c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
	}
}

// RefreshToken handles POST /user/refresh
// Matches Flask: requires refresh token, returns new {access_token}.
func RefreshToken(secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		identity := middleware.GetIdentity(c)
		if identity == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"msg": "Missing Authorization Header"})
			return
		}

		// Parse user ID from identity string
		userID, err := strconv.ParseUint(identity, 10, 64)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"msg": "Invalid token identity"})
			return
		}

		accessToken, err := middleware.GenerateAccessToken(uint(userID), secretKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error generating token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"access_token": accessToken,
		})
	}
}

// GetUser handles GET /user/:id
// Matches Flask: returns user JSON if authorized, 401 if not owner, 404 if not found.
func GetUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")
		id, err := strconv.ParseUint(idParam, 10, 64)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
			return
		}

		var user model.User
		if err := db.First(&user, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
			return
		}

		// Check ownership: only the user themselves can view their profile
		identity := middleware.GetIdentity(c)
		identityID, _ := strconv.ParseUint(identity, 10, 64)
		if uint(identityID) != user.ID {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
			return
		}

		c.JSON(http.StatusOK, toUserResponse(&user))
	}
}

// DeleteUser handles DELETE /user/:id
// Matches Flask: deletes user if authorized, returns {"message": "Deleted"}.
func DeleteUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")
		id, err := strconv.ParseUint(idParam, 10, 64)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
			return
		}

		var user model.User
		if err := db.First(&user, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
			return
		}

		// Check ownership
		identity := middleware.GetIdentity(c)
		identityID, _ := strconv.ParseUint(identity, 10, 64)
		if uint(identityID) != user.ID {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
			return
		}

		if err := db.Delete(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error deleting user"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
	}
}
