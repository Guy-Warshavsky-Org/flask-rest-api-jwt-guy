package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/modelcode-ai/flask-rest-api-jwt-guy/internal/config"
	"github.com/modelcode-ai/flask-rest-api-jwt-guy/internal/middleware"
	"github.com/modelcode-ai/flask-rest-api-jwt-guy/internal/model"
)

// UserHandler contains handlers for user endpoints.
// Mirrors Flask's app/resources/users.py.
type UserHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(db *gorm.DB, cfg *config.Config) *UserHandler {
	return &UserHandler{DB: db, Cfg: cfg}
}

// registerRequest represents the JSON body for POST /user/register.
type registerRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// userResponse mirrors the Marshmallow UserSchema output (excludes password_hash).
type userResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

// Register handles POST /user/register.
// Mirrors Flask: creates user, returns user JSON with 201,
// or 400 if user already exists.
func (h *UserHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request payload"})
		return
	}

	// Check if user already exists (mirrors: User.query.filter_by(username=...).first())
	var existing model.User
	if err := h.DB.Where("username = ?", req.Username).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "User exists"})
		return
	}

	// Create new user with hashed password.
	user := model.User{Username: req.Username}
	if err := user.SetPassword(req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to hash password"})
		return
	}

	if err := h.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create user"})
		return
	}

	// Return user JSON matching Marshmallow UserSchema output.
	c.JSON(http.StatusCreated, userResponse{
		ID:       user.ID,
		Username: user.Username,
	})
}

// loginRequest represents the JSON body for POST /user/login.
type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login handles POST /user/login.
// Mirrors Flask: verifies credentials, returns access_token and refresh_token.
func (h *UserHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid credentials"})
		return
	}

	// Find user by username.
	var user model.User
	if err := h.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid credentials"})
		return
	}

	// Verify password.
	if !user.CheckPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid credentials"})
		return
	}

	// Create tokens (identity stored as string, matching Flask's str(user.id)).
	accessToken, err := middleware.CreateAccessToken(user.ID, h.Cfg.SecretKey, h.Cfg.JWTAccessExpiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create access token"})
		return
	}

	refreshToken, err := middleware.CreateRefreshToken(user.ID, h.Cfg.SecretKey, h.Cfg.JWTRefreshExpiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create refresh token"})
		return
	}

	// Return tokens (matching Flask's jsonify(access_token=..., refresh_token=...)).
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// Logout handles POST /user/logout.
// Mirrors Flask: blacklists current access token JTI, returns {"message": "Logged out"}.
func (h *UserHandler) Logout(c *gin.Context) {
	jti, ok := middleware.GetJTI(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid token"})
		return
	}

	middleware.AddToBlacklist(jti)

	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

// Refresh handles POST /user/refresh.
// Mirrors Flask: issues a new access token from a valid refresh token.
func (h *UserHandler) Refresh(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid token"})
		return
	}

	accessToken, err := middleware.CreateAccessToken(userID, h.Cfg.SecretKey, h.Cfg.JWTAccessExpiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create access token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
	})
}

// GetUser handles GET /user/:id.
// Mirrors Flask: returns user JSON if the authenticated user matches the ID.
func (h *UserHandler) GetUser(c *gin.Context) {
	// Parse path parameter.
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	// Find user (mirrors User.query.get_or_404(id)).
	var user model.User
	if err := h.DB.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	}

	// Ownership check (mirrors: if int(get_jwt_identity()) != user.id).
	userID, _ := middleware.GetUserID(c)
	if userID != user.ID {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	c.JSON(http.StatusOK, userResponse{
		ID:       user.ID,
		Username: user.Username,
	})
}

// DeleteUser handles DELETE /user/:id.
// Mirrors Flask: deletes user (with cascade) if the authenticated user matches.
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	// Find user (mirrors User.query.get_or_404(id)).
	var user model.User
	if err := h.DB.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database error"})
		return
	}

	// Ownership check.
	userID, _ := middleware.GetUserID(c)
	if userID != user.ID {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	// Delete user. GORM cascade constraints handle related stores/items/tags.
	// Use Select to trigger GORM cascade for has-many relationships.
	if err := h.DB.Select("Stores").Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}
