package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/flask-rest-api-jwt-guy/internal/config"
	"github.com/flask-rest-api-jwt-guy/internal/middleware"
	"github.com/flask-rest-api-jwt-guy/internal/model"
)

// UserHandler holds dependencies for user-related endpoints.
type UserHandler struct {
	DB        *gorm.DB
	Config    *config.Config
	Blacklist *middleware.TokenBlacklist
}

// RegisterUserRoutes registers user routes on the given router group.
func (h *UserHandler) RegisterUserRoutes(rg *gin.RouterGroup) {
	// Public routes
	rg.POST("/register", h.Register)
	rg.POST("/login", h.Login)

	// Protected routes (access token required)
	auth := rg.Group("")
	auth.Use(middleware.AuthMiddleware(h.Config, h.Blacklist))
	auth.POST("/logout", h.Logout)
	auth.GET("/:id", h.GetUser)
	auth.DELETE("/:id", h.DeleteUser)

	// Refresh token route
	refresh := rg.Group("")
	refresh.Use(middleware.RefreshMiddleware(h.Config, h.Blacklist))
	refresh.POST("/refresh", h.Refresh)
}

type registerRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Register handles POST /user/register
func (h *UserHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}

	// Check if user already exists
	var existing model.User
	if err := h.DB.Where("username = ?", req.Username).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "User exists"})
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to hash password"})
		return
	}

	user := model.User{
		Username:     req.Username,
		PasswordHash: string(hash),
	}

	if err := h.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create user"})
		return
	}

	// Return user without password_hash (json:"-" on model handles exclusion)
	c.JSON(http.StatusCreated, gin.H{
		"id":       user.ID,
		"username": user.Username,
	})
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login handles POST /user/login
func (h *UserHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}

	var user model.User
	if err := h.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid credentials"})
		return
	}

	accessToken, err := middleware.GenerateAccessToken(user.ID, h.Config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate access token"})
		return
	}

	refreshToken, err := middleware.GenerateRefreshToken(user.ID, h.Config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// Logout handles POST /user/logout
func (h *UserHandler) Logout(c *gin.Context) {
	jti, exists := c.Get(middleware.ContextJTI)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Token JTI not found"})
		return
	}

	h.Blacklist.Add(jti.(string))

	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

// Refresh handles POST /user/refresh
func (h *UserHandler) Refresh(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid identity"})
		return
	}

	accessToken, err := middleware.GenerateAccessToken(userID, h.Config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate access token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
	})
}

// GetUser handles GET /user/:id
func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid user ID"})
		return
	}

	var user model.User
	if err := h.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	// Authorization: ensure requesting user matches
	currentUserID, err := middleware.GetUserID(c)
	if err != nil || currentUserID != user.ID {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"username": user.Username,
	})
}

// DeleteUser handles DELETE /user/:id
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid user ID"})
		return
	}

	var user model.User
	if err := h.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	// Authorization: ensure requesting user matches
	currentUserID, err := middleware.GetUserID(c)
	if err != nil || currentUserID != user.ID {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	// Delete user with cascade: delete items and tags in user's stores, then stores, then user.
	// We perform this in a transaction to ensure consistency.
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		// Get all store IDs for this user
		var storeIDs []uint
		if err := tx.Model(&model.Store{}).Where("user_id = ?", user.ID).Pluck("id", &storeIDs).Error; err != nil {
			return err
		}

		if len(storeIDs) > 0 {
			// Delete items belonging to user's stores
			if err := tx.Where("store_id IN ?", storeIDs).Delete(&model.Item{}).Error; err != nil {
				return err
			}
			// Delete tags belonging to user's stores
			if err := tx.Where("store_id IN ?", storeIDs).Delete(&model.Tag{}).Error; err != nil {
				return err
			}
			// Delete the stores
			if err := tx.Where("user_id = ?", user.ID).Delete(&model.Store{}).Error; err != nil {
				return err
			}
		}

		// Delete the user
		return tx.Delete(&user).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}
