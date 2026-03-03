package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/auth"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/config"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/db"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/middleware"
	"github.com/gwarshavsky/flask-rest-api-jwt-guy-2/internal/model"
)

// registerRequest represents the JSON body for POST /user/register.
type registerRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// loginRequest represents the JSON body for POST /user/login.
type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterUserRoutes registers all user-related endpoints on the given router group.
// Routes mirror the Flask users_bp Blueprint with url_prefix='/user'.
func RegisterUserRoutes(rg *gin.RouterGroup, cfg *config.Config) {
	user := rg.Group("/user")
	{
		// Public routes (no auth required)
		user.POST("/register", handleRegister)
		user.POST("/login", handleLogin(cfg))

		// Protected routes (require access token)
		user.POST("/logout", middleware.RequireAccessToken(cfg.JWTSecretKey), handleLogout)
		user.GET("/:id", middleware.RequireAccessToken(cfg.JWTSecretKey), handleGetUser)
		user.DELETE("/:id", middleware.RequireAccessToken(cfg.JWTSecretKey), handleDeleteUser)

		// Protected route (requires refresh token)
		user.POST("/refresh", middleware.RequireRefreshToken(cfg.JWTSecretKey), handleRefresh(cfg))
	}
}

// handleRegister implements POST /user/register.
// Creates a new user with hashed password.
// Response: 201 with {"id": N, "username": "..."} or 400 if username exists.
func handleRegister(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Missing required fields"})
		return
	}

	// Check if username already exists (mirrors Flask's User.query.filter_by)
	var existing model.User
	result := db.DB.Where("username = ?", req.Username).First(&existing)
	if result.Error == nil {
		// User found — duplicate
		c.JSON(http.StatusBadRequest, gin.H{"message": "User exists"})
		return
	}

	// Create new user
	user := model.User{Username: req.Username}
	if err := user.SetPassword(req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to hash password"})
		return
	}

	if err := db.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create user"})
		return
	}

	// Return UserResponse (excludes password_hash), matching Flask's user_schema.dump(user)
	c.JSON(http.StatusCreated, user.ToResponse())
}

// handleLogin returns a handler for POST /user/login.
// Validates credentials and returns access + refresh tokens.
// Response: 200 with {"access_token": "...", "refresh_token": "..."} or 401.
func handleLogin(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Missing required fields"})
			return
		}

		// Find user by username
		var user model.User
		result := db.DB.Where("username = ?", req.Username).First(&user)
		if result.Error != nil || !user.CheckPassword(req.Password) {
			// Matches Flask: return jsonify({'message': 'Invalid credentials'}), 401
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid credentials"})
			return
		}

		// Generate tokens with identity=str(user.id), matching Flask convention
		userIDStr := fmt.Sprintf("%d", user.ID)

		accessToken, err := auth.GenerateAccessToken(userIDStr, cfg.JWTSecretKey, cfg.JWTAccessExpires)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create access token"})
			return
		}

		refreshToken, err := auth.GenerateRefreshToken(userIDStr, cfg.JWTSecretKey, cfg.JWTRefreshExpires)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create refresh token"})
			return
		}

		// Matches Flask: return jsonify(access_token=..., refresh_token=...)
		c.JSON(http.StatusOK, gin.H{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		})
	}
}

// handleLogout implements POST /user/logout.
// Revokes the current access token by adding its JTI to the blacklist.
// Response: 200 with {"message": "Logged out"}.
func handleLogout(c *gin.Context) {
	// Get the JTI from Gin context (set by auth middleware)
	jti, exists := c.Get(middleware.ContextKeyJTI)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Token JTI not found"})
		return
	}

	// Add to blacklist (mirrors Flask: jwt_token_blacklist.add(jti))
	auth.TokenBlacklist.Add(jti.(string))

	// Matches Flask: return jsonify({'message': 'Logged out'})
	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

// handleRefresh returns a handler for POST /user/refresh.
// Accepts a valid refresh token and returns a new access token.
// Response: 200 with {"access_token": "..."}.
func handleRefresh(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the user identity from the refresh token (set by auth middleware)
		currentUser := c.GetString(middleware.ContextKeyUserID)
		if currentUser == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid token identity"})
			return
		}

		// Generate a new access token using the same identity
		// Matches Flask: create_access_token(identity=current_user)
		accessToken, err := auth.GenerateAccessToken(currentUser, cfg.JWTSecretKey, cfg.JWTAccessExpires)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create access token"})
			return
		}

		// Matches Flask: return jsonify(access_token=access_token)
		c.JSON(http.StatusOK, gin.H{
			"access_token": accessToken,
		})
	}
}

// handleGetUser implements GET /user/:id.
// Returns user data if the authenticated user matches the requested ID.
// Response: 200 with {"id": N, "username": "..."}, 401 if unauthorized, 404 if not found.
func handleGetUser(c *gin.Context) {
	// Parse the user ID from the URL parameter
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	// Find the user (mirrors Flask: User.query.get_or_404(id))
	var user model.User
	if err := db.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	// Authorization check: authenticated user must match the requested user
	// Mirrors Flask: if int(get_jwt_identity()) != user.id
	currentUserID := c.GetString(middleware.ContextKeyUserID)
	if currentUserID != fmt.Sprintf("%d", user.ID) {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	// Return user data (matches Flask: user_schema.dump(user))
	c.JSON(http.StatusOK, user.ToResponse())
}

// handleDeleteUser implements DELETE /user/:id.
// Deletes the user and all owned resources (via cascade) if the authenticated
// user matches the requested ID.
// Response: 200 with {"message": "Deleted"}, 401 if unauthorized, 404 if not found.
func handleDeleteUser(c *gin.Context) {
	// Parse the user ID from the URL parameter
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	// Find the user (mirrors Flask: User.query.get_or_404(id))
	var user model.User
	if err := db.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	// Authorization check
	currentUserID := c.GetString(middleware.ContextKeyUserID)
	if currentUserID != fmt.Sprintf("%d", user.ID) {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	// Delete user — GORM cascade should handle related stores/items/tags
	// For SQLite, we need to manually handle cascades since GORM's
	// constraint-based cascades depend on DB-level support.
	// Delete stores (and their items/tags) first, then the user.
	if err := db.DB.Where("user_id = ?", user.ID).Delete(&model.Store{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete user resources"})
		return
	}

	if err := db.DB.Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete user"})
		return
	}

	// Matches Flask: return jsonify({'message': 'Deleted'})
	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}
