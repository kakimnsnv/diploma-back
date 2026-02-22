// internal/handlers/auth.go
package handlers

import (
	"diploma-back/internal/middleware"
	"diploma-back/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) InitAuthRoutes() {
	// CORS middleware
	h.handler.Use(middleware.CORSMiddleware())

	// Public routes
	public := h.handler.Group("/api")
	{
		public.POST("/register", h.Register)
		public.POST("/login", h.Login)
		public.POST("/logout", h.Logout)
	}
}

func (h *Handler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.Services.UserService.CreateUser(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.SetCookie("auth_token", resp.Token, 86400, "/", "localhost:3000", false, true)

	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.Services.UserService.Login(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	c.SetCookie("auth_token", resp.Token, 86400, "/", "localhost:3000", false, true)

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetProfile(c *gin.Context) {
	userID := c.GetUint("userID")

	user, err := h.Services.UserService.GetProfile(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user profile"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *Handler) Logout(c *gin.Context) {
	c.SetCookie("auth_token", "", -1, "/", "localhost:3000", false, true)
	c.Status(http.StatusOK)
}
